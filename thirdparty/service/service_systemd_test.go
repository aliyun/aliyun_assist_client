package service

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

const (
	systemdScript_install = `#Version=1.0
[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}

[Service]
StandardOutput=journal+console
StandardError=journal+console
StartLimitInterval=3600
StartLimitBurst=10
ExecStart={{.Path|cmdEscape}}{{range .Arguments}} {{.|cmd}}{{end}}
{{if .ChRoot}}RootDirectory={{.ChRoot|cmd}}{{end}}
{{if .WorkingDirectory}}WorkingDirectory={{.WorkingDirectory|cmdEscape}}{{end}}
{{if .UserName}}User={{.UserName}}{{end}}
{{if .ReloadSignal}}ExecReload=/bin/kill -{{.ReloadSignal}} "$MAINPID"{{end}}
{{if .PIDFile}}PIDFile={{.PIDFile|cmd}}{{end}}
{{if and .LogOutput .HasOutputFileSupport -}}
StandardOutput=file:/var/log/{{.Name}}.out
StandardError=file:/var/log/{{.Name}}.err
{{- end}}
{{if gt .LimitNOFILE -1 }}LimitNOFILE={{.LimitNOFILE}}{{end}}
{{if .Restart}}Restart={{.Restart}}{{end}}
{{if .SuccessExitStatus}}SuccessExitStatus={{.SuccessExitStatus}}{{end}}
RestartSec=120
EnvironmentFile=-/etc/sysconfig/{{.Name}}
KillMode=process
[Install]
WantedBy=multi-user.target
`
	systemdScript_invalid = `#Version=1.0
	[Unit]
	Description={{.Description}}
	ConditionFileIsExecutable={{.Path|cmdEscape}}
	{{range $i, $dep := .Dependencies}} 
	{{$dep}} {{end}`
)

func TestGenServiceConfFile(t *testing.T) {
	systemdInst := &systemd{
		Config: &Config{
			Name: "aliyun-service",
			Option: map[string]interface{}{
				"HasOutputFileSupport": true,
				"ReloadSignal":         "HUP",
				"LimitNOFILE":          65536,
				"Restart":              "on-failure",
				"SuccessExitStatus":    "0 3",
				"LogOutput":            true,
				"SystemdScript":        systemdScript_install,
			},
		},
	}
	t.Run("CreateSuccess", func(t *testing.T) {
		symlinkPath := "/opt/aliyun-service/current"
		tempDir := t.TempDir()
		confPath := filepath.Join(tempDir, "aliyun-service.service")
		err := systemdInst.GenServiceConfFile(confPath, symlinkPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(confPath); os.IsNotExist(err) {
			t.Error("expected confPath to exist after rename, but not found")
		}

		tmpPath := confPath + ".tmp"
		if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
			t.Error("expected temp file to be renamed/deleted, but still exists")
		}

		content, err := os.ReadFile(confPath)
		if err != nil {
			t.Fatalf("failed to read generated file: %v", err)
		}

		strContent := string(content)
		tests := map[string]bool{
			"ExecStart=.../current":  strings.Contains(strContent, symlinkPath),
			"Restart=on-failure":     strings.Contains(strContent, "Restart=on-failure"),
			"SuccessExitStatus=0 3":  strings.Contains(strContent, "SuccessExitStatus=0 3"),
			"LimitNOFILE=65536":      strings.Contains(strContent, "LimitNOFILE=65536"),
			"StandardOutput=journal": strings.Contains(strContent, "StandardOutput=journal"),
			"EnvironmentFile=":       strings.Contains(strContent, "EnvironmentFile=-/etc/sysconfig/aliyun-service"),
		}

		for desc, passed := range tests {
			if !passed {
				t.Errorf("expect content to contain: %s", desc)
			}
		}
	})

	t.Run("CreateFailWithFileSyncError", func(t *testing.T) {
		defer gomonkey.ApplyMethod((*os.File)(nil), "Sync", func() error {
			return errors.New("failed to Sync file")
		}).Reset()
		tempDir := t.TempDir()
		confPath := filepath.Join(tempDir, "aliyun-service.service")
		err := systemdInst.GenServiceConfFile(confPath, "/dummy")
		if err == nil {
			t.Fatal("expected error when creating tmp file in non-existent dir, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to Sync file") {
			t.Logf("error: %v", err)
		}
	})

	t.Run("CreateFailWithConfPathNotExist", func(t *testing.T) {
		systemdInst := &systemd{}
		confPath := "/this/path/does/not/exist/and/cannot/write/aliyun-service.service"
		err := systemdInst.GenServiceConfFile(confPath, "/dummy")
		if err == nil {
			t.Fatal("expected error when creating tmp file in non-existent dir, but got nil")
		}
		if !strings.Contains(err.Error(), "open") && !os.IsPermission(err) && !os.IsNotExist(err) {
			t.Logf("error: %v", err)
		}
	})
}

func TestUpdateSymlinkPath(t *testing.T) {
	t.Run("symlink exists but points to old version", func(t *testing.T) {
		tempDir := t.TempDir()
		versionPath := filepath.Join(tempDir, "service", "v2.0.0")
		exePath := filepath.Join(versionPath, "aliyun-service")
		_ = os.MkdirAll(versionPath, 0755)
		_ = os.WriteFile(exePath, []byte("test"), 0644)
		symlinkPath := filepath.Join(tempDir, "service", "aliyun-service.symlink")
		_ = os.Symlink("v1.0.0/aliyun-service", symlinkPath)
		target, _ := os.Readlink(symlinkPath)
		assert.Equal(t, target, "v1.0.0/aliyun-service")

		symlinkPath, err := updateSymlinkPath(exePath)
		assert.Equal(t, nil, err)

		target, _ = os.Readlink(symlinkPath)
		assert.Equal(t, target, "v2.0.0/aliyun-service")
	})

	t.Run("symlink does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		versionPath := filepath.Join(tempDir, "service", "v3.0.0")
		exePath := filepath.Join(versionPath, "aliyun-service")
		_ = os.MkdirAll(versionPath, 0755)
		_ = os.WriteFile(exePath, []byte("test"), 0644)

		symlinkPath, err := updateSymlinkPath(exePath)
		assert.Equal(t, nil, err)

		target, _ := os.Readlink(symlinkPath)
		assert.Equal(t, target, "v3.0.0/aliyun-service")
	})
}

func TestInstallAndReload(t *testing.T) {
	systemdInst := &systemd{
		Config: &Config{
			Name: "aliyun",
			Option: map[string]interface{}{
				"hasOutputFileSupport": true,
				"reloadSignal":         "HUP",
				"pidFile":              "/run/app.pid",
				"limitNOFILE":          65536,
				"restart":              "on-failure",
				"successExitStatus":    "0 3",
				"logOutput":            true,
			},
		},
	}
	t.Run("NoramlCase", func(t *testing.T) {
		tempDir := t.TempDir()
		confPath := filepath.Join(tempDir, "aliyun.service")
		symlinkPath := filepath.Join(tempDir, "aliyun-service.symlink")

		calls := map[string]bool{}
		mock_runCommand := gomonkey.ApplyFunc(runCommand, func(cmd string, readStdout bool, args ...string) (int, string, error) {
			key := cmd + " " + strings.Join(args, " ")
			calls[key] = true
			return 0, "", nil
		})
		defer func() {
			mock_runCommand.Reset()
		}()
		err := systemdInst.InstallAndReload(confPath, symlinkPath, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedDaemonReload := "systemctl daemon-reload"
		expectedEnable := "systemctl enable aliyun.service"

		if !calls[expectedDaemonReload] {
			t.Errorf("expected systemctl daemon-reload to be called")
		}
		if !calls[expectedEnable] {
			t.Errorf("expected systemctl enable aliyun.service to be called")
		}
	})

	t.Run("GenServiceConfFileFail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(systemdInst), "GenServiceConfFile", func() error {
			return errors.New("failed to generate conf")
		}).Reset()

		err := systemdInst.InstallAndReload("/tmp/fake.conf", "", true)
		if err == nil {
			t.Fatal("expected error from genServiceConfFile, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to generate conf") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("DaemonReloadFail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(systemdInst), "GenServiceConfFile", func() error {
			return nil
		}).Reset()

		mock_runCommand := gomonkey.ApplyFunc(runCommand, func(cmd string, readStdout bool, args ...string) (int, string, error) {
			if args[0] != "daemon-reload" {
				return 0, "", nil
			}
			return -1, "fail", errors.New("daemon-reload failed")
		})
		defer func() {
			mock_runCommand.Reset()
		}()

		err := systemdInst.InstallAndReload("/tmp/fake.conf", "", true)
		if err == nil {
			t.Fatal("expected error from daemon-reload, but got nil")
		}
		if !strings.Contains(err.Error(), "daemon-reload failed") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("EnableFail", func(t *testing.T) {
		defer gomonkey.ApplyMethod(reflect.TypeOf(systemdInst), "GenServiceConfFile", func() error {
			return nil
		}).Reset()

		mock_runCommand := gomonkey.ApplyFunc(runCommand, func(cmd string, readStdout bool, args ...string) (int, string, error) {
			if args[0] == "enable" && args[1] == "aliyun.service" {
				return -1, "fail", errors.New("failed to enable service")
			}
			return 0, "", nil
		})
		defer func() {
			mock_runCommand.Reset()
		}()

		err := systemdInst.InstallAndReload("/tmp/fake.conf", "", true)
		if err == nil {
			t.Fatal("expected error from enable, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to enable service") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("UserServiceMode", func(t *testing.T) {
		calls := map[string]bool{}
		mock_runCommand := gomonkey.ApplyFunc(runCommand, func(cmd string, readStdout bool, args ...string) (int, string, error) {
			key := cmd + " " + strings.Join(args, " ")
			calls[key] = true
			return 0, "", nil
		})
		defer func() {
			mock_runCommand.Reset()
		}()

		systemdInst.Option["UserService"] = true
		_ = systemdInst.InstallAndReload("/tmp/test.conf", "", true)
		expectedDaemonReload := "systemctl daemon-reload --user"
		expectedEnable := "systemctl enable --user aliyun.service"

		if !calls[expectedDaemonReload] {
			t.Errorf("expected systemctl daemon-reload to be called")
		}
		if !calls[expectedEnable] {
			t.Errorf("expected systemctl enable aliyun-service.service to be called")
		}
	})
}

// test extract Version From systemd conf file
func TestExtractVersionFromContent(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name: "normal_with_template_vars",
			script: `#Version=1.0
[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}`,
			expected: "1.0",
		},
		{
			name: "version_in_middle_of_file",
			script: `[Unit]
#Version=2.1.0
Description=Something
After=network.target`,
			expected: "",
		},
		{
			name: "multiple_version_lines_take_first",
			script: `#Version=1.0
[Unit]
#Version=2.0
Description=test`,
			expected: "1.0",
		},
		{
			name: "no_version_line_at_all",
			script: `[Unit]
Description=test
After=network.target`,
			expected: "",
		},
		{
			name:     "empty_script",
			script:   "",
			expected: "",
		},
		{
			name: "version_with_spaces_around_equal",
			script: `#Version= 3.0
[Unit]`,
			expected: "3.0",
		},
		{
			name: "version_line_has_extra_comments",
			script: `#Version=1.2 # this is v1.2
[Unit]`,
			expected: "1.2 # this is v1.2",
		},
		{
			name: "template_block_does_not_contain_version_pattern",
			script: `# This is a comment
[Unit]
Description={{.Name}}
[Service]
ExecStart={{.Binary}} {{range .Args}}{{.}} {{end}}
#Version=4.0`,
			expected: "",
		},
		{
			name: "version_before_unit_section",
			script: `#Version=5.0-pre
# Generated by us
[Unit]
Description=My Service`,
			expected: "5.0-pre",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromContent(tt.script)
			if result != tt.expected {
				t.Errorf("extractVersionFromContent() = %q, want %q\nscript:\n%s", result, tt.expected, tt.script)
			}
		})
	}
}

// test file diff and if need Daemon Reload
func TestNeedDaemonReload_RealWorldUpgradeScenarios(t *testing.T) {
	oldScript := `[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}`

	newScriptWithVersion := `#Version=1.2.3
[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}`

	newerScriptWithVersion := `#Version=1.2.4
[Unit]
Description={{.Description}}
ConditionFileIsExecutable={{.Path|cmdEscape}}
{{range $i, $dep := .Dependencies}} 
{{$dep}} {{end}}`

	tests := []struct {
		name           string
		currentContent string
		newContent     string
		expected       bool
		desc           string
	}{
		{
			name:           "upgrade_from_no_version_to_versioned",
			currentContent: oldScript,
			newContent:     newScriptWithVersion,
			expected:       true,
			desc:           "首次引入 version 标记应触发 reload（功能增强）",
		},
		{
			name:           "version_increment_in_new_style",
			currentContent: newScriptWithVersion,
			newContent:     newerScriptWithVersion,
			expected:       true,
			desc:           "版本号增加，应触发 reload",
		},
		{
			name:           "same_version_in_new_style",
			currentContent: newScriptWithVersion,
			newContent:     newScriptWithVersion,
			expected:       false,
			desc:           "内容完全相同 → 无需 reload",
		},
		{
			name:           "downgrade_version",
			currentContent: newerScriptWithVersion,
			newContent:     newScriptWithVersion,
			expected:       true,
			desc:           "降级也算变更，仍需 reload",
		},
		{
			name:           "both_without_version",
			currentContent: oldScript,
			newContent:     oldScript,
			expected:       false,
			desc:           "两个都是老式脚本，无版本字段 → 无变化",
		},
		{
			name:           "new_script_has_no_version_again",
			currentContent: newScriptWithVersion,
			newContent:     oldScript,
			expected:       true,
			desc:           "新版又删了 version 字段 → 还是正常触发、可能是特定版本下需要降级",
		},
		{
			name:           "malformed_script_no_sections",
			currentContent: "#Version=1.0",
			newContent:     "#Version=2.0",
			expected:       true,
			desc:           "虽无 [Unit]，但在前5行内有 version → 视为有效变更",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkVersionDiff(tt.currentContent, tt.newContent)
			if result != tt.expected {
				t.Errorf("needDaemonReload(\n--- current ---\n%s\n--- new ---\n%s\n) = %v, want %v\n%s",
					tt.currentContent, tt.newContent, result, tt.expected, tt.desc)
			}
		})
	}
}

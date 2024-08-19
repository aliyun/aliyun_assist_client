package update

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/version"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
)

func TestRegexVersionPattern(t *testing.T) {
	regexPattern, err := regexp.Compile(regexVersionPattern)
	assert.NoErrorf(t, err, "Error should be encountered when compling regex pattern %s", regexVersionPattern)

	cases := []struct {
		in       string
		expected bool
	}{
		{"1.0.2.589", true},
		{"1", false},
		{"1.0", false},
		{"1.0.2", false},
		{"1.0.2b589", false},
		{"axt1.0.2.589", false},
		{"1.0.2.589commit08b8297c", false},
	}

	for _, c := range cases {
		if c.expected == true {
			assert.Truef(t, regexPattern.MatchString(c.in), "%s should be matched as valid version string", c.in)
		} else {
			assert.Falsef(t, regexPattern.MatchString(c.in), "%s should not be matched as valid version string", c.in)
		}
	}
}

func TestExecuteUpdateScript(t *testing.T) {
	path, _ := os.Executable()
	path, _ = filepath.Abs(filepath.Dir(path))
	filename := "script.sh"
	script := `#!/bin/bash
	echo "hello"
	`
	if runtime.GOOS == "windows" {
		filename = "script.bat"
		script = `@echo off
		echo hello`
	}
	scriptpath := filepath.Join(path, filename)
	fileutil.WriteStringToFile(scriptpath, script)
	defer func() {
		if fileutil.CheckFileIsExist(scriptpath) {
			os.Remove(scriptpath)
		}
	}()

	type args struct {
		updateScriptPath string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				updateScriptPath: scriptpath,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ExecuteUpdateScript(tt.args.updateScriptPath); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteUpdateScript() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveOldVersion(t *testing.T) {
	multiVersionPath, _ := os.Executable()
	multiVersionPath, _ = filepath.Abs(filepath.Dir(multiVersionPath))
	multiVersionPath = filepath.Join(multiVersionPath, "multiversion")
	assistVersionBackup := version.AssistVersion
	version.AssistVersion = "1.0.0.1"
	file := filepath.Join(multiVersionPath, "somefile")
	fileutil.WriteStringToFile(file, "somefile")
	sameVersionDir := filepath.Join(multiVersionPath, version.AssistVersion)
	oldVersionDir := filepath.Join(multiVersionPath, "1.0.0.0")
	otherDir := filepath.Join(multiVersionPath, "otherdir")
	pathutil.MakeSurePath(sameVersionDir)
	pathutil.MakeSurePath(oldVersionDir)
	pathutil.MakeSurePath(otherDir)
	defer func() {
		version.AssistVersion = assistVersionBackup
		os.RemoveAll(multiVersionPath)
	}()

	type args struct {
		multipleVersionDir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				multipleVersionDir: multiVersionPath,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RemoveOldVersion(tt.args.multipleVersionDir); (err != nil) != tt.wantErr {
				t.Errorf("RemoveOldVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

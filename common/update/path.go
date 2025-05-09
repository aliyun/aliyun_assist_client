package update

import (
	"path/filepath"
	"runtime"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

const (
	DefaultUnixInstallDir = "/usr/local/share/aliyun-assist"
	DefaultUnixAgentName = "aliyun-service"
	DefaultUnixUpdateScript = "update_install"
	DefaultUnixUpdatorName = "aliyun_assist_update"

	DefaultWindowsInstallDir = "C:\\ProgramData\\aliyun\\assist"
	DefaultWindowsAgentName = "aliyun_assist_service.exe"
	DefaultWindowsUpdateScript = "install.bat"
	DefaultWindowsUpdatorName = "aliyun_assist_update.exe"
)

func GetInstallDir() string {
	if installDir, err := pathutil.GetCrossVersionDir(); err == nil {
		return installDir
	}

	if runtime.GOOS == "windows" {
		return DefaultWindowsInstallDir
	}

	return DefaultUnixInstallDir
}

func GetAgentName() string {
	if runtime.GOOS == "windows" {
		return DefaultWindowsAgentName
	}

	return DefaultUnixAgentName
}

func GetAgentPathByVersion(version string) string {
	installDir := GetInstallDir()
	agentName := GetAgentName()
	return filepath.Join(installDir, version, agentName)
}

func GetUpdateScript() string {
	if runtime.GOOS == "windows" {
		return DefaultWindowsUpdateScript
	}

	return DefaultUnixUpdateScript
}

func GetUpdateScriptPathByVersion(version string) string {
	installDir := GetInstallDir()
	updateScript := GetUpdateScript()
	return filepath.Join(installDir, version, updateScript)
}

func GetUpdatorName() string {
	if runtime.GOOS == "windows" {
		return DefaultWindowsUpdatorName
	}

	return DefaultUnixUpdatorName
}

func GetUpdatorPathByCurrentProcess() string {
	dir, _ := pathutil.GetExecutableDir()
	updatorFilename := GetUpdatorName()
	updatorPath := filepath.Join(dir, updatorFilename)
	return updatorPath
}

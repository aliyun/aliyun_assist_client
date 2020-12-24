package update

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	DefaultUnixInstallDir = "/usr/local/share/aliyun-assist"
	DefaultUnixUpdateScript = "update_install"
	DefaultUnixUpdatorName = "aliyun_assist_update"

	DefaultWindowsInstallDir = "C:\\ProgramData\\aliyun\\assist"
	DefaultWindowsUpdateScript = "install.bat"
	DefaultWindowsUpdatorName = "aliyun_assist_update.exe"
)

func GetInstallDir() string {
	if runtime.GOOS == "windows" {
		return DefaultWindowsInstallDir
	}

	return DefaultUnixInstallDir
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
	path, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(path))
	updatorFilename := GetUpdatorName()
	updatorPath := filepath.Join(dir, updatorFilename)
	return updatorPath
}

package update

import (
	"os"
	"path/filepath"
	"runtime"
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
	if installDir, err := GetInstallDirByCurrentProcess(); err == nil {
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

// GetInstallDirByCurrentProcess returns normal install direcotry of agent.
// When agent is installed as **/a/b/aliyun-service, it would return **/a .
// The "normal install directory" on most Linux distribution: /usr/local/share/aliyun-assist
// on CoreOS specially: /opt/local/share/aliyun-assist
// on Windows: C:\ProgramData\aliyun\assist
func GetInstallDirByCurrentProcess() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	currentVersionDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	// Although filepath.Dir method would call filepath.Clean internally, here
	// explicitly call the method to guarantee no trailing slash in path
	cleanedCurrentVersionDir := filepath.Clean(currentVersionDir)
	multiVersionDir := filepath.Dir(cleanedCurrentVersionDir)
	return multiVersionDir, nil
}

func GetUpdatorPathByCurrentProcess() string {
	path, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(path))
	updatorFilename := GetUpdatorName()
	updatorPath := filepath.Join(dir, updatorFilename)
	return updatorPath
}

package main

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/host"
)

func PatchGolang() error {
	_, _, version, err := host.PlatformInformation()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(version, "6.1") && !strings.HasPrefix(version, "6.0") {
		return nil
	}
	exe, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(exe))
	PatchDll := path.Join(dir, "PatchGo.dll")
	_, err = syscall.LoadLibrary(PatchDll)
	return err
}

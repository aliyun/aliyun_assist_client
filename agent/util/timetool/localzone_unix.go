// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build aix darwin,!ios dragonfly freebsd linux,!android netbsd openbsd solaris

package timetool

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/yookoala/realpath"
	"golang.org/x/sys/unix"
)

var (
	// Many systems use /usr/share/zoneinfo, Solaris 2 has
	// /usr/share/lib/zoneinfo, IRIX 6 has /usr/lib/locale/TZ.
	_zoneinfoSources = []string{
		"/usr/share/zoneinfo/",
		"/usr/share/lib/zoneinfo/",
		"/usr/lib/locale/TZ/",
	}
)

// GetCurrentTimezoneName returns full name in TZ database of system timezone
// setting. On *nix systems, cases below should be taken in account:
// * When environment variable $TZ is absent, /etc/localtime should be used,
//   which is mostly a symbolic link to tzfile under sources.
// * $TZ=""
func GetCurrentTimezoneName() (string, error) {
	// consult $TZ to find the time zone to use.
	tz, ok := unix.Getenv("TZ")

	// no $TZ means use the system default /etc/localtime.
	if !ok {
		return parseLocationName("/etc/localtime")
	}

	// $TZ="" means use UTC.
	if tz == "" || tz == "UTC" || tz == ":UTC" {
		return "UTC", nil
	}

	// $TZ="foo" or $TZ=":foo" if foo is an absolute path, then the file pointed
	// by foo will be used to initialize timezone; otherwise, file
	// /usr/share/zoneinfo/foo will be used.
	if tz[0] == ':' {
		tz = tz[1:]
	}
	if tz != "" {
		if tz[0] == '/' {
			return parseLocationName(tz)
		} else if confirmLocationName(tz) {
			return tz, nil
		}
	}

	return "", ErrDetectSystemTimezoneName
}

func parseLocationName(zoneinfoPath string) (string, error) {
	realZoneinfoPath, err := realpath.Realpath(zoneinfoPath)
	if err != nil {
		return "", err
	}

	for _, source := range _zoneinfoSources {
		if !strings.HasPrefix(realZoneinfoPath, source) {
			continue
		}

		if _, err := os.Stat(realZoneinfoPath); os.IsNotExist(err) {
			continue
		}

		zoneinfoName := realZoneinfoPath[len(source):]
		return zoneinfoName, nil
	}

	return "", ErrDetectSystemTimezoneName
}

func confirmLocationName(zoneinfoName string) bool {
	for _, source := range _zoneinfoSources {
		zoneinfoPath := filepath.Join(source, zoneinfoName)
		if _, err := os.Stat(zoneinfoPath); os.IsExist(err) {
			return true
		}
	}

	return false
}

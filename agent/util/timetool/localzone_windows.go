// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timetool

import (
	"errors"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

var (
	ErrNoMatchedUnixTZName = errors.New("no matched Unix name in TZ database for Windows timezone")
)

func GetCurrentTimezoneName() (string, error) {
	var tzi windows.Timezoneinformation
	if _, err := windows.GetTimeZoneInformation(&tzi); err != nil {
		log.GetLogger().WithError(err).Warn("Failed to get timezone information on Windows, fallback to UTC time")
		return "", err
	}
	shouldBeUnixName, err := getUnixTZName(&tzi)
	if err != nil {
		log.GetLogger().WithError(err).Warn("Failed to convert Windows timezone name to Unix name in TZ database")
		return "", err
	}
	return shouldBeUnixName, nil
}

func getUnixTZName(tzi *windows.Timezoneinformation) (string, error) {
	standardName := windows.UTF16ToString(tzi.StandardName[:])
	if unixName, ok := nmaps[standardName]; ok {
		return unixName, nil
	}

	daylightName := windows.UTF16ToString(tzi.DaylightName[:])
	// As Microsoft Docs says:
	// > The StandardName and DaylightName members of the resultant
	// > TIME_ZONE_INFORMATION structure are localized according to the current
	// > user default UI language.
	// See https://docs.microsoft.com/en-us/windows/win32/api/timezoneapi/nf-timezoneapi-gettimezoneinformation
	// for details.
	englishName, err := toEnglishName(standardName, daylightName)
	if err != nil {
		return standardName, err
	}

	if unixName, ok := nmaps[englishName]; ok {
		// Cache standardName->unixName shortcut to accelerate future lookup,
		// since timezone setting should be stable for most cases
		nmaps[standardName] = unixName

		return unixName, nil
	}

	return englishName, ErrNoMatchedUnixTZName
}

// toEnglishName searches the registry for an English name of a time zone
// whose zone names are stdname and dstname and returns the English name.
func toEnglishName(stdname, dstname string) (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Time Zones`, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return "", err
	}
	for _, name := range names {
		matched, err := matchZoneKey(k, name, stdname, dstname)
		if err == nil && matched {
			return name, nil
		}
	}
	return "", errors.New(`English name for time zone "` + stdname + `" not found in registry`)
}

// matchZoneKey checks if stdname and dstname match the corresponding key
// values "MUI_Std" and MUI_Dlt" or "Std" and "Dlt" (the latter down-level
// from Vista) in the kname key stored under the open registry key zones.
func matchZoneKey(zones registry.Key, kname string, stdname, dstname string) (matched bool, err2 error) {
	k, err := registry.OpenKey(zones, kname, registry.READ)
	if err != nil {
		return false, err
	}
	defer k.Close()

	var std, dlt string
	if err = registry.LoadRegLoadMUIString(); err == nil {
		// Try MUI_Std and MUI_Dlt first, fallback to Std and Dlt if *any* error occurs
		std, err = k.GetMUIStringValue("MUI_Std")
		if err == nil {
			dlt, err = k.GetMUIStringValue("MUI_Dlt")
		}
	}
	if err != nil { // Fallback to Std and Dlt
		if std, _, err = k.GetStringValue("Std"); err != nil {
			return false, err
		}
		if dlt, _, err = k.GetStringValue("Dlt"); err != nil {
			return false, err
		}
	}

	if std != stdname {
		return false, nil
	}
	if dlt != dstname && dstname != stdname {
		return false, nil
	}
	return true, nil
}

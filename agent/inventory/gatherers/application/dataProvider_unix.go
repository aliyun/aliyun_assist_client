// +build darwin freebsd linux netbsd openbsd

package application

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/google/uuid"
)

var (
	startMarker = "<start" + randomString(8) + ">"
	endMarker   = "<end" + randomString(8) + ">"

	// rpm commands related constants
	rpmCmd                        = "rpm"
	rpmCmdArgToGetAllApplications = "-qa"
	rpmQueryFormat                = "--queryformat"
	rpmQueryFormatArgs            = `\{"Name":"` + mark(`%{NAME}`) + `","Publisher":"` + mark(`%{VENDOR}`) + `","Version":"` + mark(`%{VERSION}`) + `","Release":"` + mark(`%{RELEASE}`) + `","Epoch":"` + mark(`%{EPOCH}`) + `","InstalledTime":"` + mark(`%{INSTALLTIME}`) +
		`","ApplicationType":"` + mark(`%{GROUP}`) + `","Architecture":"` + mark(`%{ARCH}`) + `","Url":"` + mark(`%{URL}`) + `",` +
		`"Summary":"` + mark(`%{Summary}`) + `","PackageId":"` + mark(`%{SourceRPM}`) + `"\},`

	// dpkg query commands related constants
	dpkgCmd                      = "dpkg-query"
	dpkgArgsToGetAllApplications = "-W"
	dpkgQueryFormat              = `-f={"Name":"` + mark(`${Package}`) + `","Publisher":"` + mark(`${Maintainer}`) + `","Version":"` + mark(`${Version}`) + `","ApplicationType":"` + mark(`${Section}`) +
		`","Architecture":"` + mark(`${Architecture}`) + `","Url":"` + mark(`${Homepage}`) + `","Summary":"` + mark(`${Description}`) +
		// PackageId should be something like ${Filename}, but for some reason that field does not get printed,
		// so we build PackageId from parts
		`","PackageId":"` + mark(`${Package}_${Version}_${Architecture}.deb`) + `"},`

	snapPkgName                    = "snapd"
	snapCmd                        = "snap"
	snapArgsToGetAllInstalledSnaps = "list"
	snapQueryFormat                = "{\"Name\":\"%s\",\"Publisher\":\"%s\",\"Version\":\"%s\",\"ApplicationType\":\"%s\",\"Architecture\":\"%s\",\"Url\":\"%s\",\"Summary\":\"%s\",\"PackageId\":\"%s\"}"
)

func randomString(length int) string {
	return uuid.New().String()[:length]
}

func mark(s string) string {
	return startMarker + s + endMarker
}

// decoupling exec.Command for easy testability
var cmdExecutor = executeCommand

func executeCommand(command string, args ...string) ([]byte, error) {
	return exec.Command(command, args...).Output()
}

func platformInfoProvider() (name string, err error) {
	return osutil.PlatformName()
}

// collectPlatformDependentApplicationData collects all application data from the system using rpm or dpkg query.
func collectPlatformDependentApplicationData() (appData []model.ApplicationData) {

	var err error

	args := []string{dpkgArgsToGetAllApplications, dpkgQueryFormat}
	cmd := dpkgCmd
	// try dpkg first, if any error occurs, use rpm
	if appData, err = getApplicationData(cmd, args); err != nil {
		log.GetLogger().Debug("Getting applications information using dpkg failed, trying rpm now")
		cmd = rpmCmd
		args = []string{rpmCmdArgToGetAllApplications, rpmQueryFormat, rpmQueryFormatArgs}
		if appData, err = getApplicationData(cmd, args); err != nil {
			log.GetLogger().Error("collect data fail.")
			log.GetLogger().Errorf("Unable to detect package manager - hence no inventory data for %v", GathererName)
		} else {
			log.GetLogger().Debugf("collect %s data success.", GathererName)
			log.GetLogger().Debug(appData)
		}
	}
	if appData != nil && len(appData) > ApplicationCountLimit {
		err = fmt.Errorf(ApplicationCountLimitExceeded+", got %d", len(appData))
		log.GetLogger().WithError(err).Error("collection application failed")
		return []model.ApplicationData{}
	}

	// Due to ubuntu 18 use snap, so add getApplicationData here
	if snapIsInstalled(appData) {
		cmd = snapCmd
		args = []string{snapArgsToGetAllInstalledSnaps}
		var snapAppData []model.ApplicationData
		if snapAppData, err = getApplicationData(cmd, args); err != nil {
			log.GetLogger().Errorf("Getting applications information using snap failed. Skipping.")
			return
		}
		log.GetLogger().Debugf("Appending application information found using snap to application data.")
		appData = append(appData, snapAppData...)
	}
	if appData != nil && len(appData) > ApplicationCountLimit {
		err = fmt.Errorf(ApplicationCountLimitExceeded+", got %d", len(appData))
		log.GetLogger().WithError(err).Error("collection application failed")
		return []model.ApplicationData{}
	}
	return
}

func snapIsInstalled(appData []model.ApplicationData) bool {
	for _, element := range appData {
		if strings.ToLower(element.Name) == snapPkgName {
			return true
		}
	}
	return false
}

// Parse snap application data like: "Name  Version    Rev   Tracking  Publisher   Notes\n core  16-2.43.3  8689  stable    canonical*  core\n"
// into format that downstream can accept
// like: "Name":"<start4b9ad210>core<end7ca79ece>","Publisher":"<start4b9ad210>canonical*<end7ca79ece>","Version":"<start4b9ad210>16-2.43.3<end7ca79ece>"...
func parseSnapOutput(cmdOutput string) (snapOutput string) {
	var applications = strings.Split(cmdOutput, "\n")

	// last application is empty
	for i := 1; i < len(applications)-1; i++ {
		var arr = strings.Fields(applications[i])
		if len(arr) < 6 {
			log.GetLogger().Errorf("Unable get the snap list result.")
			return
		}
		var str = fmt.Sprintf(snapQueryFormat,
			mark(arr[0]),  // Name
			mark(arr[4]),  // Publisher
			mark(arr[1]),  // Version
			mark("admin"), // ApplicationType
			mark(""),      // Architecture
			mark(""),      // Url
			mark(""),      // Summary
			mark(""))      // PackageId
		snapOutput = snapOutput + str
		snapOutput = snapOutput + ","
	}
	snapOutput = strings.TrimSuffix(snapOutput, ",")
	return
}

// getApplicationData runs a shell command and gets information about all packages/applications
func getApplicationData(command string, args []string) (data []model.ApplicationData, err error) {

	/*
					Note: Following are samples of how rpm & dpkg stores package information.

					RPM:
					Name        : python27
					Version     : 2.7.10
					Release     : 4.120.amzn1
					Architecture: x86_64
					Install Date: Fri 29 Apr 2016 11:58:27 PM UTC
					Group       : Development/Languages
					Size        : 86074
					License     : Python
					Signature   : RSA/SHA256, Sat 12 Dec 2015 03:15:10 AM UTC, Key ID bcb4a85b21c0f39f
					Source RPM  : python27-2.7.10-4.120.amzn1.src.rpm
					Build Date  : Tue 08 Dec 2015 06:38:19 PM UTC
					Build Host  : build-60007.build
					Relocations : (not relocatable)
					Packager    : aliyun.com, Inc. <http://aliyun.com>
					Vendor      : aliyun.com
					URL         : http://www.python.org/
					Summary     : An interpreted, interactive, object-oriented programming language
					Description :
					Python is an interpreted, interactive, object-oriented programming
					language often compared to Tcl, Perl, Scheme or Java. Python includes
					modules, classes, exceptions, very high level dynamic data types and
					dynamic typing. Python supports interfaces to many system calls and
					libraries, as well as to various windowing systems (X11, Motif, Tk,
					Mac and MFC).

					Programmers can write new built-in modules for Python in C or C++.
					Python can be used as an extension language for applications that need
					a programmable interface.

					Note that documentation for Python is provided in the python-docs
					package.
					This package provides the "python" executable; most of the actual
					implementation is within the "python-libs" package.

					DPKG:

					Package: sed
					Essential: yes
					Priority: required
					Section: utils
					Installed-Size: 304
					Origin: Ubuntu
					Maintainer: Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>
					Bugs: https://bugs.launchpad.net/ubuntu/+filebug
					Architecture: amd64
					Multi-Arch: foreign
					Version: 4.2.2-7
					Depends: dpkg (>= 1.15.4) | install-info
					Pre-Depends: libc6 (>= 2.14), libselinux1 (>= 1.32)
					Filename: pool/main/s/sed/sed_4.2.2-7_amd64.deb
					Size: 138916
					MD5sum: cb5d3a67bb2859bc2549f1916b9a1818
					Description: The GNU sed stream editor
					Original-Maintainer: Clint Adams <clint@debian.org>
					SHA1: dc7e76d7a861b329ed73e807153c2dd89d6a0c71
					SHA256: 0623b35cdc60f8bc74e6b31ee32ed4585433fb0bc7b99c9a62985c115dbb7f0d
					Homepage: http://www.gnu.org/software/sed/
					Description-md5: 67b5a614216e15a54b09cad62d5d5afc
					Supported: 5y
					Task: minimal

		            SNAP:

		            Name: core
		            Version: 6-2.43.3
		            Rev: 8689
		            Tracking: stable
		            Publisher: anonical*
		            Notes: core

					Following fields are relevant for inventory type ACS:Application
					- Name
					- Version
				    - Release
					- Epoch
					- Publisher
					- Architecture
					- Url
					- InstalledTime
					- ApplicationType
					- Summary: For rpm, we take the multi line Description and keep the first line only.
					  The first line is a short summary. For dpkg-query we take the Summary field.
					- PackageID: we take the rpm/deb filename


					We use rpm query & dpkg-query to get above fields and then transform the data to convert into json
					to simplify its processing.

					Sample rpm query is of following format:
					rpm -qa --queryformat "\{\"Name\":\"%{NAME}\"\},"

					For more details on rpm queryformat, refer http://www.rpm.org/wiki/Docs/QueryFormat

					Sample dpkg-query is of following format:
					dpkg-query -W -f='{"Name":${binary:Package}},'

					For more details on dpkg format, refer to http://manpages.ubuntu.com/manpages/trusty/man1/dpkg-query.1.html
	*/

	var output []byte

	log.GetLogger().Debugf("Executing command: %v %v", command, args)

	if output, err = cmdExecutor(command, args...); err != nil {
		log.GetLogger().Errorf("Failed to execute command : %v %v with error - %v",
			command,
			args,
			err.Error())
		log.GetLogger().Debugf("Command Stderr: %v", string(output))
		err = fmt.Errorf("Command failed with error: %v", string(output))
	} else {
		cmdOutput := string(output)
		// parse snap result
		if command == "snap" {
			cmdOutput = parseSnapOutput(cmdOutput)
		}
		log.GetLogger().Debugf("Command output: %v", cmdOutput)

		if data, err = convertToApplicationData(cmdOutput); err != nil {
			err = fmt.Errorf("Unable to convert query output to ApplicationData - %v", err.Error())
			log.GetLogger().Errorf(err.Error())
			log.GetLogger().Debugf("Command output: %s", cmdOutput)
		} else {
			log.GetLogger().Debug(data)
			log.GetLogger().Debugf("Number of applications detected - %v", len(data))
		}
	}

	return
}

// convertToApplicationData converts query output into json string so that it can be deserialized easily
func convertToApplicationData(input string) (data []model.ApplicationData, err error) {

	//This implementation is closely tied to the kind of rpm/dpkg query. A change in query MUST be accompanied
	//with a change in transform logic or else json formatting will be impacted.

	/*
		Sample format of our rpm queryformat & dpkg format:
		rpm -qa --queryformat "\{\"Name\":\"%{NAME}\"\},"
		dpkg-query -W -f='{"Name":${binary:Package}},'

		Above queries will generate data in following format:
		{"Name":"nss-softokn"},{"Name":"basesystem"},{"Name":"pcre"},

		Keeping above sample in mind - we do following operations:
		- remove trailing white spaces
		- remove trailing ','
		- prefix '[' at the beginning & ']' at the end

		After above operation above sample data will convert to:
		[{"Name":"nss-softokn"},{"Name":"basesystem"},{"Name":"pcre"}]
	*/

	str := convertEntriesToJsonArray(input)
	// keep single line out of multi-line fields and escape special characters
	str, err = replaceMarkedFields(str, startMarker, endMarker, cleanupJSONField)
	if err != nil {
		return
	}

	//unmarshal json string accordingly.
	if err = json.Unmarshal([]byte(str), &data); err == nil {

		//transform the date & architecture - by iterating over all elements
		for i, item := range data {
			if item.InstalledTime != "" {
				if sec, err := strconv.ParseInt(item.InstalledTime, 10, 64); err == nil {
					//InstalledTime must comply with format: 2016-07-30T18:15:37Z to provide better search experience for customers
					tm := time.Unix(sec, 0).UTC()
					item.InstalledTime = tm.Format(time.RFC3339)
				}
				//ignore the date transformation if error is encountered
			}

			/*
				dpkg reports applications architecture as amd64, i386, all
				rpm reports applications architecture as x86_64, i386, noarch

				For consistency, we want to ensure that architecture is reported as x86_64, i386 for
				64bit & 32bit applications across all platforms.
			*/
			item.Architecture = model.FormatArchitecture(item.Architecture)

			/*
					Especially for rpm packages:
					Package Id should be like: n-e:v-r.a or n-v-r.a (n: name; e: epoch; v: version, r: release; a: architecture)
					If there is a : in the package Id string, everything before it is the epoch. If not, omit the epoch.
				    Refer to: https://www.redhat.com/archives/rpm-list/2000-October/msg00075.html
			*/
			if item.Epoch == "(none)" {
				if strings.Contains(item.PackageId, ":") {
					//nameEpoch: name-epoch
					var nameEpoch string = strings.Split(item.PackageId, ":")[0]
					item.Epoch = strings.Split(nameEpoch, "-")[1]
				} else {
					item.Epoch = ""
				}
			}

			data[i] = item
		}
	}

	return
}

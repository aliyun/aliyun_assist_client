package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

type filterObj struct {
	Path         string
	Pattern      []string
	Recursive    bool
	DirScanLimit *int
}

type fileInfoObject struct {
	fi   os.FileInfo
	path string
}

var getFullPath func(path string, mapping func(string) string) (string, error)

// Limits to help keep file information under item size limit and prevent long scanning.
// The Dir Limits can be configured through input parameters
const FileCountLimit = 500
const FileCountLimitExceeded = "File Count Limit Exceeded"
const DirScanLimit = 5000
const DirScanLimitExceeded = "Directory Scan Limit Exceeded"

var DirScanLimitError = errors.New(DirScanLimitExceeded)
var FileCountLimitError = errors.New(FileCountLimitExceeded)

var readDirFunc = readDir
var getFilesFunc = getFiles
var existsPath = exists
var filepathWalk = filepath.Walk
var getMetaDataFunc = getMetaData

// readDir is a wrapper on ioutil.ReadDir for easy testability
func readDir(dirname string) ([]os.FileInfo, error) {
	return ioutil.ReadDir(dirname)
}

//removeDuplicates deduplicates the input array of model.FileData
func removeDuplicatesFileData(elements []model.FileData) (result []model.FileData) {
	// Use map to record duplicates as we find them.
	encountered := map[model.FileData]bool{}
	for v := range elements {
		if !encountered[elements[v]] {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

//removeDuplicatesString deduplicates array of strings
func removeDuplicatesString(elements []string) (result []string) {
	encountered := map[string]bool{}
	for _, element := range elements {
		if !encountered[element] {
			encountered[element] = true
			result = append(result, element)
		}
	}
	return result
}

//exists check if the file path exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getFiles(path string, pattern []string, recursive bool, fileLimit int, dirLimit int) (validFiles []string, err error) {
	var ex bool
	ex, err = existsPath(path)
	if err != nil {
		log.GetLogger().Error(err)
		return
	}
	if !ex {
		log.GetLogger().Error(fmt.Errorf("Path %v does not exist!", path))
		return
	}
	dirScanCount := 0
	if recursive {
		err = filepathWalk(path, func(fp string, fi os.FileInfo, err error) error {
			if err != nil {
				log.GetLogger().Error(err)
				return nil
			}
			if fi.IsDir() {
				dirScanCount++
				if dirScanCount > dirLimit {
					log.GetLogger().Errorf("Scanned maximum allowed directories. Returning collected files")
					return DirScanLimitError
				}
				return nil

			}
			if fileMatchesAnyPattern(pattern, fi.Name()) {
				validFiles = append(validFiles, fp)
				if len(validFiles) > fileLimit {
					log.GetLogger().Errorf("Found more than limit of %d files", FileCountLimit)
					return FileCountLimitError
				}
			}
			return nil
		})
	} else {
		files, readDirErr := readDirFunc(path)
		if readDirErr != nil {
			log.GetLogger().Error(readDirErr)
			err = readDirErr
			return
		}

		dirScanCount++
		for _, fi := range files {
			if fi.IsDir() {
				continue
			}
			if fileMatchesAnyPattern(pattern, fi.Name()) {
				validFiles = append(validFiles, filepath.Join(path, fi.Name()))
				if len(validFiles) > fileLimit {
					log.GetLogger().Errorf("Found more than limit of %d files", FileCountLimit)
					err = FileCountLimitError
					return
				}
			}
		}

	}

	log.GetLogger().Debugf("DirScanned %d", dirScanCount)
	return
}

//getAllMeta processes the filter, gets paths of all filtered files, and get file info of all files
func getAllMeta(config model.Config) (data []model.FileData, err error) {
	jsonBody := []byte(strings.Replace(config.Filters, `\`, `/`, -1)) //this is to convert the backslash in windows path to slash
	var filterList []filterObj
	if err = json.Unmarshal(jsonBody, &filterList); err != nil {
		log.GetLogger().Error(err)
		return
	}
	var fileList []string
	for _, filter := range filterList {

		var fullPath string
		var getPathErr error
		var dirScanLimit int
		if fullPath, getPathErr = getFullPath(filter.Path, os.Getenv); getPathErr != nil {
			log.GetLogger().Error(getPathErr)
			continue
		}
		fileLimit := FileCountLimit - len(fileList)
		if filter.DirScanLimit == nil {
			dirScanLimit = DirScanLimit
		} else {
			dirScanLimit = *filter.DirScanLimit
		}
		log.GetLogger().Debugf("Dir Scan Limit %d", dirScanLimit)
		foundFiles, getFilesErr := getFilesFunc(fullPath, filter.Pattern, filter.Recursive, fileLimit, dirScanLimit)
		// We should only break, if we get limit error, otherwise we should continue collecting other data
		if getFilesErr != nil {
			log.GetLogger().Error(getFilesErr)
			if getFilesErr == FileCountLimitError || getFilesErr == DirScanLimitError {
				return nil, getFilesErr
			}
		}
		fileList = append(fileList, foundFiles...)
		fileList = removeDuplicatesString(fileList)
	}

	if len(fileList) > 0 {
		data, err = getMetaDataFunc(fileList)
	}
	log.GetLogger().Debugf("Collected Files %d", len(data))
	return
}

//fileMatchesAnyPattern returns true if file name matches any pattern specified
func fileMatchesAnyPattern(pattern []string, fname string) bool {
	for _, item := range pattern {
		matched, matchErr := filepath.Match(item, fname)
		if matchErr != nil {
			log.GetLogger().Error(matchErr)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

//collectFileData returns a list of file information based on the given configuration
func collectFileData(config model.Config) (data []model.FileData, err error) {
	getFullPath = expand
	data, err = getAllMeta(config)
	log.GetLogger().WithError(err).Debugf("collected %d file data", len(data))
	return
}

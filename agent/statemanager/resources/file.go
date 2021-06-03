package resources

import (
	"encoding/json"
)

type FileState struct {
	Ensure          string
	State           string
	DestinationPath string
	Mode            string
	Owner           string
	Group           string
	SourcePath      string
	Contents        string
	Checksum        string
	Attributes      string
}

func (fs *FileState) Load(properties map[string]interface{}) (err error) {
	data, err := json.Marshal(properties)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, fs)
	return
}

func (fs *FileState) Apply() (status string, extraInfo string, err error) {
	return Compliant, "", nil
}

func (fs *FileState) Monitor() (status string, extraInfo string, err error) {
	return Compliant, "", nil
}

package models

type SendFileTaskInfo struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
	Destination string `json:"destination"`
	Group       string `json:"group"`
	Mode        string `json:"mode"`
	Name        string `json:"name"`
	Overwrite   bool   `json:"overwrite"`
	Owner       string `json:"owner"`
	Signature   string `json:"signature"`
	TaskID      string `json:"taskID"`
	Timeout     int64  `json:"timeout"`
	Output      OutputInfo
}

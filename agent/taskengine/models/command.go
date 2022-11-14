package models

type RunTaskRepeatType string

const (
	RunTaskOnce           RunTaskRepeatType = "Once"
	RunTaskCron           RunTaskRepeatType = "Period"
	RunTaskNextRebootOnly RunTaskRepeatType = "NextRebootOnly"
	RunTaskEveryReboot    RunTaskRepeatType = "EveryReboot"
	RunTaskRate           RunTaskRepeatType = "Rate"
	RunTaskAt             RunTaskRepeatType = "At"
)

type OutputInfo struct {
	Interval  int  `json:"interval"`
	LogQuota  int  `json:"logQuota"`
	SkipEmpty bool `json:"skipEmpty"`
	SendStart bool `json:"sendStart"`
}

type RunTaskInfo struct {
	InstanceId      string `json:"instanceId"`
	CommandType     string `json:"type"`
	TaskId          string `json:"taskID"`
	CommandId       string `json:"commandId"`
	EnableParameter bool   `json:"enableParameter"`
	TimeOut         string `json:"timeOut"`
	CommandName     string `json:"commandName"`
	Content         string `json:"commandContent"`
	WorkingDir      string `json:"workingDirectory"`
	Args            string `json:"args"`
	Cronat          string `json:"cron"`
	Username        string `json:"username"`
	Password        string `json:"windowsPasswordName"`
	CreationTime    int64 `json:"creationTime"`
	ContainerId     string `json:"containerId"`
	ContainerName   string `json:"containerName"`
	BuiltinParameters map[string]string `json:"builtInParameter"`

	Output          OutputInfo
	Repeat          RunTaskRepeatType
}

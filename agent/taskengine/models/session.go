package models

type SessionTaskInfo struct {
	CmdContent   string `json:"cmdContent"`
	Username     string `json:"username"`
	Password     string `json:"windowsPasswordName"`
	SessionId    string `json:"channelId"`
	WebsocketUrl string `json:"websocketUrl"`
	PortNumber  string `json:"portNumber"`
	FlowLimit	 int    `json:"flowLimit"` // 最大流量 单位 bps
}

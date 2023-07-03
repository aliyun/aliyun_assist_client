package pluginmanager

// 调用一下接口后需要主动向服务端上报插件状态
var (
	NEED_REFRESH_STATUS_API []string = []string{"--install", "--uninstall", "--start", "--stop", "--upgrade", "--restart"}
)

// 向服务端请求插件列表的接口数据
type PluginListRequest struct {
	OsType     string `json:"osType"`
	PluginName string `json:"pluginName"`
	Version    string `json:"version"`
	Arch       string `json:"arch"`
}
type PluginListResponse struct {
	Code       int          `json:"code"`
	RequestId  string       `json:"requestId"`
	InstanceId string       `json:"instanceId"`
	PluginList []PluginInfo `json:"pluginList"`
}

// 状态上报的请求数据
type PluginStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
}
type PluginStatusResquest struct {
	Plugin []PluginStatus `json:"plugin"`
}

const (
	NORMAL_REPORT = 0
	LAZY_REPORT   = 1 // 懒上报

	// 限制上报时插件名和插件版本字段的长度
	PLUGIN_NAME_MAXLEN    = 255
	PLUGIN_VERSION_MAXLEN = 32
)

// 状态上报的响应数据
type PluginStatusResponse struct {
	InstanceId      string `json:"instanceId"`
	Code            int    `json:"code"`
	ScanInterval    int    `json:"scanInterval"`    // 下次Agent周期扫描插件状态的频率（兜底），单位秒
	PullInterval    int    `json:"pullInterval"`    // 下次Agent主动拉取插件状态的频率，单位秒
	RefreshInterval int    `json:"refreshInterval"` // 周期扫描到Failed状态的常驻插件后会拉起，在RefreshInterval间隔后上报插件状态
	ReportType      int    `json:"reportType"`      // 上报方式 0-正常上报；1-懒上报
}

// PluginUpdateCheck 检查升级请求数据
type PluginUpdateCheck struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
}
type PluginUpdateCheckRequest struct {
	Os     string              `json:"os"`
	Arch   string              `json:"arch"`
	Plugin []PluginUpdateCheck `json:"plugin"`
}

// PluginUpdateCheckResp 检查升级的响应数据
type PluginUpdateCheckResponse struct {
	InstanceId   string             `json:"instanceId"`
	NextInterval int                `json:"nextInterval"`
	Plugin       []PluginUpdateInfo `json:"plugin"`
}

// PluginUpdateInfo 升级插件的信息
type PluginUpdateInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Timeout int `json:"timeout"`
}

package pluginmanager

// 插件状态
const (
	// PluginTypeHealth 健康 0
	PERSIST_RUNNING string = "PERSIST_RUNNING"
	// PluginTypeFail 未成功启动 1
	PERSIST_FAIL	string = "PERSIST_FAIL"
	// PluginUnknown 未知
	PERSIST_UNKNOWN string = "PERSIST_UNKNOWN"

	// PluginTypeOnce 一次性插件已安装 2
	ONCE_INSTALLED	string = "ONCE_INSTALLED"
)

// 插件类型
const (
	// PluginOneTime 一次性插件
	PluginOneTime int = 0
	// PluginPersist 常驻型插件
	PluginPersist int = 1
)

type PluginInfo struct {
	PluginID       string `json:"pluginId"`
	Name           string `json:"name"`
	Arch           string `json:"arch"`
	OSType         string `json:"osType"`
	Version        string `json:"version"`
	Publisher      string `json:"publisher"`
	Url            string `json:"url"`
	Md5            string `json:"md5"`
	RunPath        string `json:"runPath"`
	Timeout        string `json:"timeout"`
	IsPreInstalled string `json:"isPreInstalled"`
	PluginType     int    `json:"pluginType"`
}

// InstalledPlugins 和installed_plugins文件内容一致，用于解析json
type InstalledPlugins struct {
	PluginList []PluginInfo `json:"pluginList"`
}

// PluginStatus 用于解析常驻插件的--status接口的输出
type PluginStatusJson struct {
	PluginID string `json:"pluginId"`
	Name     string `json:"name"`
	Status   int    `json:"status"`
	Version  string `json:"version"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}



type PluginStatus struct {
	PluginID string `json:"pluginId"`
	Name string `json:"name"`
	Status string `json:"status"`
	Version string `json:"version"`
	Uptime int `json:"uptime"` // 插件运行时间 单位秒
}
type PluginStatusResquest struct {
	Os string `json:"os"`
	Arch string `json:"arch"`
	Plugin []PluginStatus `json:"plugin"`
}
// 状态上报的响应数据
type PluginStatusResponse struct {
	InstanceId string `json:"instanceId"`
	NextInterval int `json:"nextInterval"`
}



// PluginUpdateCheck 检查升级请求数据
type PluginUpdateCheck struct {
	PluginID string `json:"pluginId"`
	Name string `json:"name"`
	Version  string `json:"version"`
}
type PluginUpdateCheckRequest struct {
	Os string `json:"os"`
	Arch string `json:"arch"`
	Plugin []PluginUpdateCheck `json:"plugin"`
}

// PluginUpdateCheckResp 检查升级的响应数据
type PluginUpdateCheckResponse struct {
	InstanceId string `json:"instanceId"`
	NextInterval int `json:"nextInterval"`
	Plugin []PluginUpdateInfo `json:"plugin"`
}
// PluginUpdateInfo 升级插件的信息
type PluginUpdateInfo struct {
	NeedUpdate int `json:"needUpdate"`
	Info struct {
		PluginID   string `json:"pluginId"`
		Name       string `json:"name"`
		Url        string `json:"url"`
		Md5        string `json:"md5"`
		RunPath    string `json:"runPath"`
		TimeOut    string `json:"timeout"`
		Version    string `json:"version"`
	} `json:"info"`
	
}

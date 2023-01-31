package pluginmanager

// 插件状态
const (
	// PluginTypeHealth 健康 0
	PERSIST_RUNNING string = "PERSIST_RUNNING"
	// PluginTypeFail 未成功启动 1
	PERSIST_FAIL string = "PERSIST_FAIL"
	// PluginUnknown 未知
	PERSIST_UNKNOWN string = "PERSIST_UNKNOWN"

	// PluginTypeOnce 一次性插件已安装 2
	ONCE_INSTALLED string = "ONCE_INSTALLED"
	// 已删除
	REMOVED string = "REMOVED"
)

// 插件类型
const (
	// PluginOneTime 一次性插件
	PLUGIN_ONCE     string = "Once"
	PLUGIN_ONCE_INT int    = 0
	// PluginPersist 常驻型插件
	PLUGIN_PERSIST     string = "Persist"
	PLUGIN_PERSIST_INT int    = 1
	// PluginUnknown 未知类型
	PLUGIN_UNKNOWN     string = "Unknown"
	PLUGIN_UNKNOWN_INT int    = -1
)

// 调用一下接口后需要主动向服务端上报插件状态
var (
	NEED_REFRESH_STATUS_API []string = []string{"--install", "--uninstall", "--start", "--stop", "--upgrade", "--restart"}
)

type PluginInfo struct {
	PluginID          string      `json:"pluginId"`
	Name              string      `json:"name"`
	Arch              string      `json:"arch"`
	OSType            string      `json:"osType"`
	Version           string      `json:"version"`
	Publisher         string      `json:"publisher"`
	Url               string      `json:"url"`
	Md5               string      `json:"md5"`
	RunPath           string      `json:"runPath"`
	Timeout           string      `json:"timeout"`
	IsPreInstalled    string      `json:"isPreInstalled"`
	PluginType_        interface{} `json:"pluginType"`
	pluginTypeStr     string
	HeartbeatInterval int  `json:"heartbeatInterval"`
	IsRemoved         bool `json:"isRemoved"`
}

func (pi *PluginInfo) PluginType() string {
	if pi.pluginTypeStr == "" {
		switch pi.PluginType_.(type) {
		case string:
			pt, _ := pi.PluginType_.(string)
			if pt == PLUGIN_ONCE  || pt == "" { // 空字符串
				pi.pluginTypeStr = PLUGIN_ONCE
			} else if pt == PLUGIN_PERSIST {
				pi.pluginTypeStr = PLUGIN_PERSIST
			} else {
				pi.pluginTypeStr = PLUGIN_UNKNOWN
			}
		case float64:
			pt, _ := pi.PluginType_.(float64)
			if pt == float64(PLUGIN_ONCE_INT) {
				pi.pluginTypeStr = PLUGIN_ONCE
			} else if pt == float64(PLUGIN_PERSIST_INT) {
				pi.pluginTypeStr = PLUGIN_PERSIST
			} else {
				pi.pluginTypeStr = PLUGIN_UNKNOWN
			}
		case nil: // 字段不存在
			pi.pluginTypeStr = PLUGIN_ONCE
		default:
			pi.pluginTypeStr = PLUGIN_UNKNOWN
		}
		// 先注释掉，避免把installed_plugins中pluginType字段替换成string类型后，旧版本的acs-plugin-manager不识别
		// pi.pluginType = pi.pluginTypeStr
	}
	return pi.pluginTypeStr
}

func (pi *PluginInfo) SetPluginType(pluginType string) {
	pi.pluginTypeStr = pluginType
	// 兼容旧版本，installed_plugins文件中的pluginType字段仍然使用int类型
	if pluginType == PLUGIN_PERSIST {
		pi.PluginType_ = PLUGIN_PERSIST_INT
	} else {
		pi.PluginType_ = PLUGIN_ONCE_INT
	}
}

// InstalledPlugins 和installed_plugins文件内容一致，用于解析json
type InstalledPlugins struct {
	PluginList []PluginInfo `json:"pluginList"`
}

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
	PluginID string `json:"pluginId"`
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
	NeedUpdate int `json:"needUpdate"`
	Info       struct {
		PluginID string `json:"pluginId"`
		Name     string `json:"name"`
		Url      string `json:"url"`
		Md5      string `json:"md5"`
		RunPath  string `json:"runPath"`
		TimeOut  string `json:"timeout"`
		Version  string `json:"version"`
	} `json:"info"`
}

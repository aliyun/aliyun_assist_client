package pluginmanager

const (
	ARCH_64      = "x64"
	ARCH_32      = "x86"
	ARCH_ARM     = "arm"
	ARCH_UNKNOWN = "unknown"
)

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
	// Commander 云助手组件
	PLUGIN_COMMANDER     string = "Commander"
	PLUGIN_COMMANDER_INT        = 2
	// PluginUnknown 未知类型
	PLUGIN_UNKNOWN     string = "Unknown"
	PLUGIN_UNKNOWN_INT int    = -1
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
	PluginType_       interface{} `json:"pluginType"`
	pluginTypeStr     string
	HeartbeatInterval int           `json:"heartbeatInterval"`
	IsRemoved         bool          `json:"isRemoved"`
	AddSysTag         bool          `json:"addSysTag"`
	CommanderInfo     CommanderInfo `json:"commanderInfo"`
}

type CommanderInfo struct {
	PidFile      string `json:"pidFile"`
	EndpointType string `json:"endpointType"`
	EndpointFile string `json:"endpointFile"`
	ApiVersion   string `json:"apiVersion"`
}

func (pi *PluginInfo) PluginType() string {
	if pi.pluginTypeStr == "" {
		switch pi.PluginType_.(type) {
		case string:
			pt, _ := pi.PluginType_.(string)
			if pt == PLUGIN_ONCE || pt == "" { // 空字符串
				pi.pluginTypeStr = PLUGIN_ONCE
			} else if pt == PLUGIN_PERSIST {
				pi.pluginTypeStr = PLUGIN_PERSIST
			} else if pt == PLUGIN_COMMANDER {
				pi.pluginTypeStr = PLUGIN_COMMANDER
			} else {
				pi.pluginTypeStr = PLUGIN_UNKNOWN
			}
		case float64:
			pt, _ := pi.PluginType_.(float64)
			if pt == float64(PLUGIN_ONCE_INT) {
				pi.pluginTypeStr = PLUGIN_ONCE
			} else if pt == float64(PLUGIN_PERSIST_INT) {
				pi.pluginTypeStr = PLUGIN_PERSIST
			} else if pt == float64(PLUGIN_COMMANDER_INT) {
				pi.pluginTypeStr = PLUGIN_COMMANDER
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
	} else if pluginType == PLUGIN_COMMANDER {
		pi.PluginType_ = PLUGIN_COMMANDER_INT
	} else {
		pi.PluginType_ = PLUGIN_ONCE_INT
	}
}

package pluginmodel

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

type PluginStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
	SysTagType string `json:"sysTagType,omitempty"`
}

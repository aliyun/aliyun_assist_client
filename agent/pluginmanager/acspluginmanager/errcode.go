package acspluginmanager

// 退出码
const (
	SUCCESS = 0

	//
	CHECK_ENDPOINT_FAIL         = 233 // check end point fail
	PACKAGE_NOT_FOUND           = 234 // 插件包未找到
	PACKAGE_FORMAT_ERR         = 235 // 插件包的格式错误，不是zip格式
	UNZIP_ERR                   = 236 // 解压插件包时错误
	UNMARSHAL_ERR               = 237 // 解析json文件时错误（config.json）
	PLUGIN_FORMAT_ERR           = 238 // 插件格式错误，如config.json或者插件可执行文件缺失，插件与当前系统平台不适配等
	MD5_CHECK_FAIL              = 239 // MD5校验失败
	DOWNLOAD_FAIL               = 240 // 下载失败
	LOAD_INSTALLEDPLUGINS_ERR   = 241 // 读取 installed_plugins文件错误
	DUMP_INSTALLEDPLUGINS_ERR   = 242 // 保存内容到 installed_plugins文件错误
	GET_ONLINE_PACKAGE_INFO_ERR = 243 // 获取线上的插件包信息时错误
	EXECUTABLE_PERMISSION_ERR   = 244 // linux下赋予脚本可执行权限时错误
	REMOVE_FILE_ERR             = 245 // 删除文件时错误
	EXECUTE_FAILED              = 246 // 执行插件失败
	EXECUTE_TIMEOUT             = 247 // 执行超时
)

var (
	ErrorStrMap = map[int]string{
		CHECK_ENDPOINT_FAIL:         "CHECK_ENDPOINT_FAIL",
		PACKAGE_NOT_FOUND:           "PACKAGE_NOT_FOUND",           // 插件包未找到
		PACKAGE_FORMAT_ERR:         "PACKAGE_FORMAT_ERR",          // 插件包的格式错误，不是zip格式
		UNZIP_ERR:                   "UNZIP_ERR",                   // 解压插件包时错误
		UNMARSHAL_ERR:               "UNMARSHAL_ERR",               // 解析json文件时错误（config.json）
		PLUGIN_FORMAT_ERR:           "PLUGIN_FORMAT_ERR",           // 插件格式错误，如config.json或者插件可执行文件缺失，插件与当前系统平台不适配等
		MD5_CHECK_FAIL:              "MD5_CHECK_FAIL",              // MD5校验失败
		DOWNLOAD_FAIL:               "DOWNLOAD_FAIL",               // 下载失败
		LOAD_INSTALLEDPLUGINS_ERR:   "LOAD_INSTALLEDPLUGINS_ERR",   // 读取 installed_plugins文件错误
		DUMP_INSTALLEDPLUGINS_ERR:   "DUMP_INSTALLEDPLUGINS_ERR",   // 保存内容到 installed_plugins文件错误
		GET_ONLINE_PACKAGE_INFO_ERR: "GET_ONLINE_PACKAGE_INFO_ERR", // 获取线上的插件包信息时错误
		EXECUTABLE_PERMISSION_ERR:   "EXECUTABLE_PERMISSION_ERR",
		REMOVE_FILE_ERR:             "REMOVE_FILE_ERR", // 删除文件时报错
		EXECUTE_FAILED:              "EXECUTE_FAILED_ERR",
		EXECUTE_TIMEOUT:             "EXECUTE_TIMEOUT_ERR",
	}
)

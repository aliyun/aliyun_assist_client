package acspluginmanager

// 退出码
const (
	SUCCESS = 0

	//
	CHECK_ENDPOINT_FAIL         = 233 // check end point fail
	PACKAGE_NOT_FOUND           = 234 // 插件包未找到
	PACKAGE_FORMART_ERR         = 235 // 插件包的格式错误，不是zip格式
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

const (
	CHECK_ENDPOINT_FAIL_STR         = "CHECK_ENDPOINT_FAIL: "
	PACKAGE_NOT_FOUND_STR           = "PACKAGE_NOT_FOUND: "           // 插件包未找到
	PACKAGE_FORMART_ERR_STR         = "PACKAGE_FORMAT_ERR: "          // 插件包的格式错误，不是zip格式
	UNZIP_ERR_STR                   = "UNZIP_ERR: "                   // 解压插件包时错误
	UNMARSHAL_ERR_STR               = "UNMARSHAL_ERR: "               // 解析json文件时错误（config.json）
	PLUGIN_FORMAT_ERR_STR           = "PLUGIN_FORMAT_ERR: "           // 插件格式错误，如config.json或者插件可执行文件缺失，插件与当前系统平台不适配等
	MD5_CHECK_FAIL_STR              = "MD5_CHECK_FAIL: "              // MD5校验失败
	DOWNLOAD_FAIL_STR               = "DOWNLOAD_FAIL: "               // 下载失败
	LOAD_INSTALLEDPLUGINS_ERR_STR   = "LOAD_INSTALLEDPLUGINS_ERR: "   // 读取 installed_plugins文件错误
	DUMP_INSTALLEDPLUGINS_ERR_STR   = "DUMP_INSTALLEDPLUGINS_ERR: "   // 保存内容到 installed_plugins文件错误
	GET_ONLINE_PACKAGE_INFO_ERR_STR = "GET_ONLINE_PACKAGE_INFO_ERR: " // 获取线上的插件包信息时错误
	EXECUTABLE_PERMISSION_ERR_STR   = "EXECUTABLE_PERMISSION_ERR: "
	REMOVE_FILE_ERR_STR             = "REMOVE_FILE_ERR: " // 删除文件时报错
	EXECUTE_FAILED_STR              = "EXECUTE_FAILED_ERR: "
	EXECUTE_TIMEOUT_STR             = "EXECUTE_TIMEOUT_ERR: "
)

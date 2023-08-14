package acspluginmanager

import (
	"errors"
	"fmt"
)

// 退出码
const (
	SUCCESS = 0

	//
	LOCKING_ERR                 = 232 // Failed to manipulate concurrent locking
	CHECK_ENDPOINT_FAIL         = 233 // check end point fail
	PACKAGE_NOT_FOUND           = 234 // 插件包未找到
	PACKAGE_FORMAT_ERR          = 235 // 插件包的格式错误，不是zip格式
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
		LOCKING_ERR:                 "LOCKING_ERR",
		CHECK_ENDPOINT_FAIL:         "CHECK_ENDPOINT_FAIL",
		PACKAGE_NOT_FOUND:           "PACKAGE_NOT_FOUND",           // 插件包未找到
		PACKAGE_FORMAT_ERR:          "PACKAGE_FORMAT_ERR",          // 插件包的格式错误，不是zip格式
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

	ErrPackageNotFound = errors.New("Could not found package")
)

type ExitingError interface {
	Error() string

	ExitCode() int

	Unwrap() error
}

type _exitingError struct {
	exitCode int
	cause error
	message string
}

func (e *_exitingError) Error() string {
	return e.message
}

func (e *_exitingError) ExitCode() int {
	return e.exitCode
}

func (e *_exitingError) Unwrap() error {
	return e.cause
}

func NewPackageNotFoundExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: PACKAGE_NOT_FOUND,
		cause: cause,
		message: message,
	}
}

func NewUnzipExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: UNZIP_ERR,
		cause: cause,
		message: message,
	}
}

func NewUnmarshalExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: UNMARSHAL_ERR,
		cause: cause,
		message: message,
	}
}

func NewPluginFormatExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: PLUGIN_FORMAT_ERR,
		cause: cause,
		message: message,
	}
}

func NewMD5CheckExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: MD5_CHECK_FAIL,
		cause: cause,
		message: message,
	}
}

func NewDownloadExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: DOWNLOAD_FAIL,
		cause: cause,
		message: message,
	}
}

func NewLoadInstalledPluginsExitingError(cause error) ExitingError {
	return &_exitingError{
		exitCode: LOAD_INSTALLEDPLUGINS_ERR,
		cause: cause,
		message: "Load installed_plugins err: "+cause.Error(),
	}
}

func NewDumpInstalledPluginsExitingError(cause error) ExitingError {
	return &_exitingError{
		exitCode: DUMP_INSTALLEDPLUGINS_ERR,
		cause: cause,
		message: "Update installed_plugins file err: "+cause.Error(),
	}
}

func NewGetOnlinePackageInfoExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: GET_ONLINE_PACKAGE_INFO_ERR,
		cause: cause,
		message: message,
	}
}

func NewExecutablePermissionExitingError(cause error, message string) ExitingError {
	return &_exitingError{
		exitCode: EXECUTABLE_PERMISSION_ERR,
		cause: cause,
		message: message,
	}
}

type OpenPluginLockFileError struct {
	cause error
}

func NewOpenPluginLockFileError(cause error) *OpenPluginLockFileError {
	return &OpenPluginLockFileError{
		cause: cause,
	}
}

func (e *OpenPluginLockFileError) Error() string {
	return fmt.Sprintf("Failed to open or create plugin-wise lock file: %v", e.cause)
}

func (e *OpenPluginLockFileError) Unwrap() error {
	return e.cause
}

type AcquirePluginSharedLockError struct {
	cause error
}

func NewAcquirePluginSharedLockError(cause error) *AcquirePluginSharedLockError {
	return &AcquirePluginSharedLockError{
		cause: cause,
	}
}

func (e *AcquirePluginSharedLockError) Error() string {
	return fmt.Sprintf("Failed to acquire plugin-wise shared lock: %v", e.cause)
}

func (e *AcquirePluginSharedLockError) Unwrap() error {
	return e.cause
}

type AcquirePluginExclusiveLockError struct {
	cause error
}

func NewAcquirePluginExclusiveLockError(cause error) *AcquirePluginExclusiveLockError {
	return &AcquirePluginExclusiveLockError{
		cause: cause,
	}
}

func (e *AcquirePluginExclusiveLockError) Error() string {
	return fmt.Sprintf("Failed to acquire plugin-wise exclusive lock: %v", e.cause)
}

func (e *AcquirePluginExclusiveLockError) Unwrap() error {
	return e.cause
}

type OpenPluginVersionLockFileError struct {
	cause error
}

func NewOpenPluginVersionLockFileError(cause error) *OpenPluginVersionLockFileError {
	return &OpenPluginVersionLockFileError{
		cause: cause,
	}
}

func (e *OpenPluginVersionLockFileError) Error() string {
	return fmt.Sprintf("Failed to open or create plugin-version-wise lock file: %v", e.cause)
}

func (e *OpenPluginVersionLockFileError) Unwrap() error {
	return e.cause
}

type AcquirePluginVersionSharedLockError struct {
	cause error
}

func NewAcquirePluginVersionSharedLockError(cause error) *AcquirePluginVersionSharedLockError {
	return &AcquirePluginVersionSharedLockError{
		cause: cause,
	}
}

func (e *AcquirePluginVersionSharedLockError) Error() string {
	return fmt.Sprintf("Failed to acquire plugin-version-wise shared lock: %v", e.cause)
}

func (e *AcquirePluginVersionSharedLockError) Unwrap() error {
	return e.cause
}

type AcquirePluginVersionExclusiveLockError struct {
	cause error
}

func NewAcquirePluginVersionExclusiveLockError(cause error) *AcquirePluginVersionExclusiveLockError {
	return &AcquirePluginVersionExclusiveLockError{
		cause: cause,
	}
}

func (e *AcquirePluginVersionExclusiveLockError) Error() string {
	return fmt.Sprintf("Failed to acquire plugin-version-wise exclusive lock: %v", e.cause)
}

func (e *AcquirePluginVersionExclusiveLockError) Unwrap() error {
	return e.cause
}

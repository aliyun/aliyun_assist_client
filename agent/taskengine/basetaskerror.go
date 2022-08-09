package taskengine

// presetWrapErrorCode defines and MUST contain all error codes that will be reported
// as failure
type presetWrapErrorCode int

const (
	// Positive value is reserved for syscall errno on *nix and API error code
	// on Windows

	wrapErrGetScriptPathFailed presetWrapErrorCode = -(1 + iota)
	wrapErrUnknownCommandType
	wrapErrBase64DecodeFailed
	wrapErrSaveScriptFileFailed
	wrapErrSetExecutablePermissionFailed
	wrapErrSetWindowsPermissionFailed
	wrapErrExecuteScriptFailed
	wrapErrNoEnoughSpace
	wrapErrScriptFileExisted
	wrapErrPowershellNotFound
	wrapErrSystemDefaultShellNotFound
	wrapErrResolveEnvironmentParameterFailed
)

var (
	presetErrorPrefixes = map[presetWrapErrorCode]string{
		wrapErrGetScriptPathFailed: "GetScriptPathFailed",
		wrapErrUnknownCommandType: "UnknownCommandType",
		wrapErrBase64DecodeFailed: "Base64DecodeFailed",
		wrapErrSaveScriptFileFailed: "SaveScriptFileFailed",
		wrapErrSetExecutablePermissionFailed: "SetExecutablePermissionFailed",
		wrapErrSetWindowsPermissionFailed: "SetWindowsPermissionFailed",
		wrapErrExecuteScriptFailed: "ExecuteScriptFailed",
		wrapErrNoEnoughSpace: "NoEnoughSpace",
		wrapErrScriptFileExisted: "ScriptFileExisted",
		wrapErrPowershellNotFound: "PowershellNotFound",
		wrapErrSystemDefaultShellNotFound: "SystemDefaultShellNotFound",
	}
)

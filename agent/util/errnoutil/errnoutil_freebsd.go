package errnoutil

import (
	"syscall"
	"errors"
)

var (
	ErrnoPhrases = map[syscall.Errno]string{
		syscall.EPERM:   "OperationNotPermitted",   // //1
		syscall.ENOENT:  "NoSuchFileOrDirectory",   // //2
		syscall.ESRCH:   "NoSuchProcess",           // //3
		syscall.EINTR:   "InterruptedSystemCall",   // //4
		syscall.EIO:     "InputOutputError",        // //5
		syscall.ENXIO:   "NoSuchDeviceOrAddress",   // //6
		syscall.E2BIG:   "ArgumentListTooLong",     // //7
		syscall.ENOEXEC: "ExecFormatError",         // //8
		syscall.EBADF:   "BadFileDescriptor",       // //9
		syscall.ECHILD:  "NoChildProcesses",        // //10
		syscall.EDEADLK: "ResourceDeadlockAvoided", // //11
		/* 11 was EAGAIN */
		syscall.ENOMEM:  "CannotAllocateMemory",        // //12
		syscall.EACCES:  "PermissionDenied",            // //13
		syscall.EFAULT:  "BadAddress",                  // //14
		syscall.ENOTBLK: "BlockDeviceRequired",         // //15
		syscall.EBUSY:   "DeviceOrResourceBusy",        // //16
		syscall.EEXIST:  "FileExists",                  // //17
		syscall.EXDEV:   "InvalidCrossDeviceLink",      // //18
		syscall.ENODEV:  "NoSuchDevice",                // //19
		syscall.ENOTDIR: "NotADirectory",               // //20
		syscall.EISDIR:  "IsADirectory",                // //21
		syscall.EINVAL:  "InvalidArgument",             // //22
		syscall.ENFILE:  "TooManyOpenFilesInSystem",    // //23
		syscall.EMFILE:  "TooManyOpenFiles",            // //24
		syscall.ENOTTY:  "InappropriateIoctlForDevice", // //25
		syscall.ETXTBSY: "TextFileBusy",                // //26
		syscall.EFBIG:   "FileTooLarge",                // //27
		syscall.ENOSPC:  "NoSpaceLeftOnDevice",         // //28
		syscall.ESPIPE:  "IllegalSeek",                 // //29
		syscall.EROFS:   "ReadOnlyFileSystem",          // //30
		syscall.EMLINK:  "TooManyLinks",                // //31
		syscall.EPIPE:   "BrokenPipe",                  // //32

		/* math software */
		syscall.EDOM:   "NumericalArgumentOutOfDomain", // //33
		syscall.ERANGE: "NumericalResultOutOfRange",    // //34

		/* non-blocking and interrupt i/o */
		syscall.EAGAIN: "ResourceTemporarilyUnavailable", // //35
		// syscall.EWOULDBLOCK: "Operation would block",            //EAGAIN
		syscall.EINPROGRESS: "OperationNowInProgress",     // //36
		syscall.EALREADY:    "OperationAlreadyInProgress", // //37

		/* ipc/network software -- argument errors */
		syscall.ENOTSOCK:        "SocketOperationOnNonSocket", // //38
		syscall.EDESTADDRREQ:    "DestinationAddressRequired", // //39
		syscall.EMSGSIZE:        "MessageTooLong",             // //40
		syscall.EPROTOTYPE:      "ProtocolWrongTypeForSocket", // //41
		syscall.ENOPROTOOPT:     "ProtocolNotAvailable",       // //42
		syscall.EPROTONOSUPPORT: "ProtocolNotSupported",       // //43
		syscall.ESOCKTNOSUPPORT: "SocketTypeNotSupported",     // //44
		// syscall.EOPNOTSUPP:      "Operation not supported",                         // //45
		syscall.ENOTSUP:       "OperationNotSupported",               // EOPNOTSUPP
		syscall.EPFNOSUPPORT:  "ProtocolFamilyNotSupported",          // //46
		syscall.EAFNOSUPPORT:  "AddressFamilyNotSupportedByProtocol", // //47
		syscall.EADDRINUSE:    "AddressAlreadyInUse",                 // //48
		syscall.EADDRNOTAVAIL: "CannotAssignRequestedAddress",        // //49

		/* ipc/network software -- operational errors */
		syscall.ENETDOWN:     "NetworkIsDown",                            // //50
		syscall.ENETUNREACH:  "NetworkIsUnreachable",                     // //51
		syscall.ENETRESET:    "NetworkDroppedConnectionOnReset",          // //52
		syscall.ECONNABORTED: "SoftwareCausedConnectionAbort",            // //53
		syscall.ECONNRESET:   "ConnectionResetByPeer",                    // //54
		syscall.ENOBUFS:      "NoBufferSpaceAvailable",                   // //55
		syscall.EISCONN:      "TransportEndpointIsAlreadyConnected",      // //56
		syscall.ENOTCONN:     "TransportEndpointIsNotConnected",          // //57
		syscall.ESHUTDOWN:    "CannotSendAfterTransportEndpointShutdown", // //58
		syscall.ETOOMANYREFS: "TooManyReferencesCannotSplice",            // //59
		syscall.ETIMEDOUT:    "ConnectionTimedOut",                       // //60
		syscall.ECONNREFUSED: "ConnectionRefused",                        // //61

		syscall.ELOOP:         "TooManyLevelsOfSymbolicLinks",                // //62
		syscall.ENAMETOOLONG:  "FileNameTooLong",                             // //63
		syscall.EHOSTDOWN:     "HostIsDown",                                  // //64
		syscall.EHOSTUNREACH:  "NoRouteToHost",                               // //65
		syscall.ENOTEMPTY:     "DirectoryNotEmpty",                           // //66
		syscall.EPROCLIM:      "TooManyProcesses",                            // //67
		syscall.EUSERS:        "TooManyUsers",                                // //68
		syscall.EDQUOT:        "DiskQuotaExceeded",                           // //69
		syscall.ESTALE:        "StaleFileHandle",                             // //70
		syscall.EREMOTE:       "ObjectIsRemote",                              // //71
		syscall.EBADRPC:       "RpcStructIsBad",                              // //72
		syscall.ERPCMISMATCH:  "RpcVersionWrong",                             // //73
		syscall.EPROGUNAVAIL:  "RpcProg.NotAvail",                            // //74
		syscall.EPROGMISMATCH: "ProgramVersionWrong",                         // //75
		syscall.EPROCUNAVAIL:  "BadProcedureForProgram",                      // //76
		syscall.ENOLCK:        "NoLocksAvailable",                            // //77
		syscall.ENOSYS:        "FunctionNotImplemented",                      // //78
		syscall.EFTYPE:        "InappropriateFileTypeOrFormat",               // //79
		syscall.EAUTH:         "AuthenticationError",                         // //80
		syscall.ENEEDAUTH:     "NeedAuthenticator",                           // //81
		syscall.EIDRM:         "IdentifierRemoved",                           // //82
		syscall.ENOMSG:        "NoMessageOfDesiredType",                      // //83
		syscall.EOVERFLOW:     "ValueTooLargeForDefinedDataType",             // //84
		syscall.ECANCELED:     "OperationCanceled",                           // //85
		syscall.EILSEQ:        "InvalidOrIncompleteMultibyteOrWideCharacter", // //86
		syscall.ENOATTR:       "AttributeNotFound",                           // //87

		syscall.EDOOFUS:         "ProgrammingError",             // //88
		syscall.EBADMSG:         "BadMessage",                   // //89
		syscall.EMULTIHOP:       "MultihopAttempted",            // //90
		syscall.ENOLINK:         "LinkHasBeenSevered",           // //91
		syscall.EPROTO:          "ProtocolError",                // //92
		syscall.ENOTCAPABLE:     "CapabilitiesInsufficient",     // //93
		syscall.ECAPMODE:        "NotPermittedInCapabilityMode", // //94
		syscall.ENOTRECOVERABLE: "StateNotRecoverable",          // //95
		syscall.EOWNERDEAD:      "OwnerDied",                    // //96
		// syscall.EINTEGRITY: "Integrity check failed", // //97
		// syscall.ELAST: "Must be equal largest errno", // //97

	}
)

// IsNoEnoughSpaceError detects "no space left on device" error
func IsNoEnoughSpaceError(err error) bool {
	return errors.Is(err, syscall.ENOSPC)
}

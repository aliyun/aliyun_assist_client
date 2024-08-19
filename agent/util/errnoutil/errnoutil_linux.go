package errnoutil

import (
	"syscall"
	"errors"
)

var (
	ErrnoPhrases = map[syscall.Errno]string{
		syscall.E2BIG:           "ArgumentListTooLong",                         // 0x7  argument list too long
		syscall.EACCES:          "PermissionDenied",                            // 0xd  permission denied
		syscall.EADDRINUSE:      "AddressAlreadyInUse",                         // 0x62  address already in use
		syscall.EADDRNOTAVAIL:   "CannotAssignRequestedAddress",                // 0x63  cannot assign requested address
		syscall.EADV:            "AdvertiseError",                              // 0x44  advertise error
		syscall.EAFNOSUPPORT:    "AddressFamilyNotSupportedByProtocol",         // 0x61  address family not supported by protocol
		syscall.EAGAIN:          "ResourceTemporarilyUnavailable",              // 0xb  resource temporarily unavailable
		syscall.EALREADY:        "OperationAlreadyInProgress",                  // 0x72  operation already in progress
		syscall.EBADE:           "InvalidExchange",                             // 0x34  invalid exchange
		syscall.EBADF:           "BadFileDescriptor",                           // 0x9  bad file descriptor
		syscall.EBADFD:          "FileDescriptorInBadState",                    // 0x4d  file descriptor in bad state
		syscall.EBADMSG:         "BadMessage",                                  // 0x4a  bad message
		syscall.EBADR:           "InvalidRequestDescriptor",                    // 0x35  invalid request descriptor
		syscall.EBADRQC:         "InvalidRequestCode",                          // 0x38  invalid request code
		syscall.EBADSLT:         "InvalidSlot",                                 // 0x39  invalid slot
		syscall.EBFONT:          "BadFontFileFormat",                           // 0x3b  bad font file format
		syscall.EBUSY:           "DeviceOrResourceBusy",                        // 0x10  device or resource busy
		syscall.ECANCELED:       "OperationCanceled",                           // 0x7d  operation canceled
		syscall.ECHILD:          "NoChildProcesses",                            // 0xa  no child processes
		syscall.ECHRNG:          "ChannelNumberOutOfRange",                     // 0x2c  channel number out of range
		syscall.ECOMM:           "CommunicationErrorOnSend",                    // 0x46  communication error on send
		syscall.ECONNABORTED:    "SoftwareCausedConnectionAbort",               // 0x67  software caused connection abort
		syscall.ECONNREFUSED:    "ConnectionRefused",                           // 0x6f  connection refused
		syscall.ECONNRESET:      "ConnectionResetByPeer",                       // 0x68  connection reset by peer
		syscall.EDEADLK:         "ResourceDeadlockAvoided",                     // 0x23  resource deadlock avoided
		syscall.EDESTADDRREQ:    "DestinationAddressRequired",                  // 0x59  destination address required
		syscall.EDOM:            "NumericalArgumentOutOfDomain",                // 0x21  numerical argument out of domain
		syscall.EDOTDOT:         "RfsSpecificError",                            // 0x49  RFS specific error
		syscall.EDQUOT:          "DiskQuotaExceeded",                           // 0x7a  disk quota exceeded
		syscall.EEXIST:          "FileExists",                                  // 0x11  file exists
		syscall.EFAULT:          "BadAddress",                                  // 0xe  bad address
		syscall.EFBIG:           "FileTooLarge",                                // 0x1b  file too large
		syscall.EHOSTDOWN:       "HostIsDown",                                  // 0x70  host is down
		syscall.EHOSTUNREACH:    "NoRouteToHost",                               // 0x71  no route to host
		syscall.EIDRM:           "IdentifierRemoved",                           // 0x2b  identifier removed
		syscall.EILSEQ:          "InvalidOrIncompleteMultibyteOrWideCharacter", // 0x54  invalid or incomplete multibyte or wide character
		syscall.EINPROGRESS:     "OperationNowInProgress",                      // 0x73  operation now in progress
		syscall.EINTR:           "InterruptedSystemCall",                       // 0x4  interrupted system call
		syscall.EINVAL:          "InvalidArgument",                             // 0x16  invalid argument
		syscall.EIO:             "InputOutputError",                            // 0x5  input/output error
		syscall.EISCONN:         "TransportEndpointIsAlreadyConnected",         // 0x6a  transport endpoint is already connected
		syscall.EISDIR:          "IsADirectory",                                // 0x15  is a directory
		syscall.EISNAM:          "IsANamedTypeFile",                            // 0x78  is a named type file
		syscall.EKEYEXPIRED:     "KeyHasExpired",                               // 0x7f  key has expired
		syscall.EKEYREJECTED:    "KeyWasRejectedByService",                     // 0x81  key was rejected by service
		syscall.EKEYREVOKED:     "KeyHasBeenRevoked",                           // 0x80  key has been revoked
		syscall.EL2HLT:          "Level2Halted",                                // 0x33  level 2 halted
		syscall.EL2NSYNC:        "Level2NotSynchronized",                       // 0x2d  level 2 not synchronized
		syscall.EL3HLT:          "Level3Halted",                                // 0x2e  level 3 halted
		syscall.EL3RST:          "Level3Reset",                                 // 0x2f  level 3 reset
		syscall.ELIBACC:         "CanNotAccessANeededSharedLibrary",            // 0x4f  can not access a needed shared library
		syscall.ELIBBAD:         "AccessingACorruptedSharedLibrary",            // 0x50  accessing a corrupted shared library
		syscall.ELIBEXEC:        "CannotExecASharedLibraryDirectly",            // 0x53  cannot exec a shared library directly
		syscall.ELIBMAX:         "AttemptingToLinkInTooManySharedLibraries",    // 0x52  attempting to link in too many shared libraries
		syscall.ELIBSCN:         "LibSectionInAoutCorrupted",                   // 0x51  .lib section in a.out corrupted
		syscall.ELNRNG:          "LinkNumberOutOfRange",                        // 0x30  link number out of range
		syscall.ELOOP:           "TooManyLevelsOfSymbolicLinks",                // 0x28  too many levels of symbolic links
		syscall.EMEDIUMTYPE:     "WrongMediumType",                             // 0x7c  wrong medium type
		syscall.EMFILE:          "TooManyOpenFiles",                            // 0x18  too many open files
		syscall.EMLINK:          "TooManyLinks",                                // 0x1f  too many links
		syscall.EMSGSIZE:        "MessageTooLong",                              // 0x5a  message too long
		syscall.EMULTIHOP:       "MultihopAttempted",                           // 0x48  multihop attempted
		syscall.ENAMETOOLONG:    "FileNameTooLong",                             // 0x24  file name too long
		syscall.ENAVAIL:         "NoXenixSemaphoresAvailable",                  // 0x77  no XENIX semaphores available
		syscall.ENETDOWN:        "NetworkIsDown",                               // 0x64  network is down
		syscall.ENETRESET:       "NetworkDroppedConnectionOnReset",             // 0x66  network dropped connection on reset
		syscall.ENETUNREACH:     "NetworkIsUnreachable",                        // 0x65  network is unreachable
		syscall.ENFILE:          "TooManyOpenFilesInSystem",                    // 0x17  too many open files in system
		syscall.ENOANO:          "NoAnode",                                     // 0x37  no anode
		syscall.ENOBUFS:         "NoBufferSpaceAvailable",                      // 0x69  no buffer space available
		syscall.ENOCSI:          "NoCsiStructureAvailable",                     // 0x32  no CSI structure available
		syscall.ENODATA:         "NoDataAvailable",                             // 0x3d  no data available
		syscall.ENODEV:          "NoSuchDevice",                                // 0x13  no such device
		syscall.ENOENT:          "NoSuchFileOrDirectory",                       // 0x2  no such file or directory
		syscall.ENOEXEC:         "ExecFormatError",                             // 0x8  exec format error
		syscall.ENOKEY:          "RequiredKeyNotAvailable",                     // 0x7e  required key not available
		syscall.ENOLCK:          "NoLocksAvailable",                            // 0x25  no locks available
		syscall.ENOLINK:         "LinkHasBeenSevered",                          // 0x43  link has been severed
		syscall.ENOMEDIUM:       "NoMediumFound",                               // 0x7b  no medium found
		syscall.ENOMEM:          "CannotAllocateMemory",                        // 0xc  cannot allocate memory
		syscall.ENOMSG:          "NoMessageOfDesiredType",                      // 0x2a  no message of desired type
		syscall.ENONET:          "MachineIsNotOnTheNetwork",                    // 0x40  machine is not on the network
		syscall.ENOPKG:          "PackageNotInstalled",                         // 0x41  package not installed
		syscall.ENOPROTOOPT:     "ProtocolNotAvailable",                        // 0x5c  protocol not available
		syscall.ENOSPC:          "NoSpaceLeftOnDevice",                         // 0x1c  no space left on device
		syscall.ENOSR:           "OutOfStreamsResources",                       // 0x3f  out of streams resources
		syscall.ENOSTR:          "DeviceNotAStream",                            // 0x3c  device not a stream
		syscall.ENOSYS:          "FunctionNotImplemented",                      // 0x26  function not implemented
		syscall.ENOTBLK:         "BlockDeviceRequired",                         // 0xf  block device required
		syscall.ENOTCONN:        "TransportEndpointIsNotConnected",             // 0x6b  transport endpoint is not connected
		syscall.ENOTDIR:         "NotADirectory",                               // 0x14  not a directory
		syscall.ENOTEMPTY:       "DirectoryNotEmpty",                           // 0x27  directory not empty
		syscall.ENOTNAM:         "NotAXenixNamedTypeFile",                      // 0x76  not a XENIX named type file
		syscall.ENOTRECOVERABLE: "StateNotRecoverable",                         // 0x83  state not recoverable
		syscall.ENOTSOCK:        "SocketOperationOnNonSocket",                  // 0x58  socket operation on non-socket
		syscall.ENOTSUP:         "OperationNotSupported",                       // 0x5f  operation not supported
		syscall.ENOTTY:          "InappropriateIoctlForDevice",                 // 0x19  inappropriate ioctl for device
		syscall.ENOTUNIQ:        "NameNotUniqueOnNetwork",                      // 0x4c  name not unique on network
		syscall.ENXIO:           "NoSuchDeviceOrAddress",                       // 0x6  no such device or address
		syscall.EOVERFLOW:       "ValueTooLargeForDefinedDataType",             // 0x4b  value too large for defined data type
		syscall.EOWNERDEAD:      "OwnerDied",                                   // 0x82  owner died
		syscall.EPERM:           "OperationNotPermitted",                       // 0x1  operation not permitted
		syscall.EPFNOSUPPORT:    "ProtocolFamilyNotSupported",                  // 0x60  protocol family not supported
		syscall.EPIPE:           "BrokenPipe",                                  // 0x20  broken pipe
		syscall.EPROTO:          "ProtocolError",                               // 0x47  protocol error
		syscall.EPROTONOSUPPORT: "ProtocolNotSupported",                        // 0x5d  protocol not supported
		syscall.EPROTOTYPE:      "ProtocolWrongTypeForSocket",                  // 0x5b  protocol wrong type for socket
		syscall.ERANGE:          "NumericalResultOutOfRange",                   // 0x22  numerical result out of range
		syscall.EREMCHG:         "RemoteAddressChanged",                        // 0x4e  remote address changed
		syscall.EREMOTE:         "ObjectIsRemote",                              // 0x42  object is remote
		syscall.EREMOTEIO:       "RemoteIOError",                               // 0x79  remote I/O error
		syscall.ERESTART:        "InterruptedSystemCallShouldBeRestarted",      // 0x55  interrupted system call should be restarted
		syscall.ERFKILL:         "OperationNotPossibleDueToRfKill",             // 0x84  operation not possible due to RF-kill
		syscall.EROFS:           "ReadOnlyFileSystem",                          // 0x1e  read-only file system
		syscall.ESHUTDOWN:       "CannotSendAfterTransportEndpointShutdown",    // 0x6c  cannot send after transport endpoint shutdown
		syscall.ESOCKTNOSUPPORT: "SocketTypeNotSupported",                      // 0x5e  socket type not supported
		syscall.ESPIPE:          "IllegalSeek",                                 // 0x1d  illegal seek
		syscall.ESRCH:           "NoSuchProcess",                               // 0x3  no such process
		syscall.ESRMNT:          "SrmountError",                                // 0x45  srmount error
		syscall.ESTALE:          "StaleFileHandle",                             // 0x74  stale file handle
		syscall.ESTRPIPE:        "StreamsPipeError",                            // 0x56  streams pipe error
		syscall.ETIME:           "TimerExpired",                                // 0x3e  timer expired
		syscall.ETIMEDOUT:       "ConnectionTimedOut",                          // 0x6e  connection timed out
		syscall.ETOOMANYREFS:    "TooManyReferencesCannotSplice",               // 0x6d  too many references: cannot splice
		syscall.ETXTBSY:         "TextFileBusy",                                // 0x1a  text file busy
		syscall.EUCLEAN:         "StructureNeedsCleaning",                      // 0x75  structure needs cleaning
		syscall.EUNATCH:         "ProtocolDriverNotAttached",                   // 0x31  protocol driver not attached
		syscall.EUSERS:          "TooManyUsers",                                // 0x57  too many users
		syscall.EXDEV:           "InvalidCrossDeviceLink",                      // 0x12  invalid cross-device link
		syscall.EXFULL:          "ExchangeFull",                                // 0x36  exchange full
	}
)

// IsNoEnoughSpaceError detects "no space left on device" error
func IsNoEnoughSpaceError(err error) bool {
	return errors.Is(err, syscall.ENOSPC)
}

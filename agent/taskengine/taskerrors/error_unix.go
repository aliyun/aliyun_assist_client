//go:build linux || freebsd
// +build linux freebsd

package taskerrors

import (
	"errors"
	"strings"
	"syscall"
)

var (
	errnoPhrases = map[syscall.Errno]string{
		syscall.EPERM:        "OperationNotPermitted",          // 0x1
		syscall.ENOENT:       "NoSuchFileOrDirectory",          // 0x2
		syscall.EIO:          "InputOutputError",               // 0x5
		syscall.E2BIG:        "ArgumentListTooLong",            // 0x7
		syscall.ENOEXEC:      "ExecFormatError",                // 0x8
		syscall.EBADF:        "BadFileDescriptor",              // 0x9
		syscall.EAGAIN:       "ResourceTemporarilyUnavailable", // 0xb
		syscall.ENOMEM:       "CannotAllocateMemory",           // 0xc
		syscall.EACCES:       "PermissionDenied",               // 0xd
		syscall.EFAULT:       "BadAddress",                     // 0xe
		syscall.EEXIST:       "FileExists",                     // 0x11
		syscall.ENOTDIR:      "NotADirectory",                  // 0x14
		syscall.EISDIR:       "IsADirectory",                   // 0x15
		syscall.EINVAL:       "InvalidArgument",                // 0x16
		syscall.ENFILE:       "TooManyOpenFilesInSystem",       // 0x17
		syscall.EMFILE:       "TooManyOpenFiles",               // 0x18
		syscall.ETXTBSY:      "TextFileBusy",                   // 0x1a
		syscall.ENOSPC:       "NoEnoughSpace",                  // 0x1c
		syscall.EROFS:        "ReadonlyFileSystem",             // 0x1e
		syscall.EMLINK:       "TooManyLinks",                   // 0x1f
		syscall.ENAMETOOLONG: "FileNameTooLong",                // 0x24
		syscall.ELOOP:        "TooManySymbolicLinkLevels",      // 0x28
		syscall.EDQUOT:       "DiskQuotaExceeded",              // 0x7a
	}
)

func (e *baseError) Error() string {
	errcodePhrase := e.category
	var errno syscall.Errno
	if errors.As(e.cause, &errno) {
		if errnoPhrase, ok := errnoPhrases[errno]; ok {
			errcodePhrase += "." + errnoPhrase
		}
	}

	messages := []string{errcodePhrase}
	if e.Description != "" {
		messages = append(messages, e.Description)
	}
	if e.cause != nil {
		messages = append(messages, e.cause.Error())
	}

	return strings.Join(messages, ": ")
}

func (e *baseError) Code() ErrorCode {
	var errno syscall.Errno
	if errors.As(e.cause, &errno) {
		return ErrorCode(errno)
	}

	return e.categoryCode
}

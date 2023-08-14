package filelock

// This package provides general shared/exclusive locking mechanism on files,
// which is heavily based on existing code from battle-tested third-party
// libraries.
//
// See link below for details about the reference prototype of filelock:
// https://pkg.go.dev/cmd/go/internal/lockedfile/internal/filelock
//
// For original implementation of internal functions flock() and funlock(),
// please see link below:
// https://github.com/etcd-io/bbolt/blob/master/bolt_unix.go
// https://github.com/etcd-io/bbolt/blob/master/bolt_windows.go

import (
	"time"
)

const (
	flockRetryTimeout = 50 * time.Millisecond
)

// A File provides the minimal set of methods required to lock an open file.
// File implementations must be usable as map keys.
// The usual implementation is *os.File.
type File interface {
	// Fd returns a valid file descriptor.
	// (If the File is an *os.File, it must not be closed.)
	Fd() uintptr
}

// IsLockingError checks whether an error object is os-specific error which
// represents failure to acquire lock specified
func IsLockingError(err error) bool {
	return isLockingError(err)
}

// Lock places an advisory write lock on the file, blocking until it can be
// locked or timing out.
//
// If Lock returns nil, no other process will be able to place a read or write
// lock on the file until this process exits, closes f, or calls Unlock on it.
//
// If f's descriptor is already read- or write-locked, the behavior of Lock is
// unspecified.
//
// Closing the file may or may not release the lock promptly. Callers should
// ensure that Unlock is always called when Lock succeeds.
func Lock(f File, timeout time.Duration) error {
	return flock(f, true, timeout)
}

// RLock places an advisory read lock on the file, blocking until it can be
// locked or timing out.
//
// If RLock returns nil, no other process will be able to place a write lock on
// the file until this process exits, closes f, or calls Unlock on it.
//
// If f is already read- or write-locked, the behavior of RLock is unspecified.
//
// Closing the file may or may not release the lock promptly. Callers should
// ensure that Unlock is always called if RLock succeeds.
func RLock(f File, timeout time.Duration) error {
	return flock(f, false, timeout)
}

// TryLock places an advisory write lock on the file, but returns immediately
// when it can not be locked.
//
// If Lock returns nil, no other process will be able to place a read or write
// lock on the file until this process exits, closes f, or calls Unlock on it.
//
// If f's descriptor is already read- or write-locked, the behavior of Lock is
// unspecified.
//
// Closing the file may or may not release the lock promptly. Callers should
// ensure that Unlock is always called when Lock succeeds.
func TryLock(f File) error {
	return tryFlock(f, true)
}

// TryRLock places an advisory read lock on the file, but returns immediately
// when it can not be locked.
//
// If RLock returns nil, no other process will be able to place a write lock on
// the file until this process exits, closes f, or calls Unlock on it.
//
// If f is already read- or write-locked, the behavior of RLock is unspecified.
//
// Closing the file may or may not release the lock promptly. Callers should
// ensure that Unlock is always called if RLock succeeds.
func TryRLock(f File) error {
	return tryFlock(f, false)
}

// Unlock removes an advisory lock placed on f by this process.
//
// The caller must not attempt to unlock a file that is not locked.
func Unlock(f File) error {
	return funlock(f)
}

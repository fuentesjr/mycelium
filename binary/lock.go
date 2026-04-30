package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// lockFileName is the mount-level lock file. It sits under the reserved
// '_'-prefix so agents cannot write it; the file content is irrelevant
// (it stays empty), so reads of it are harmless.
const lockFileName = "_lock"

// acquireMountLock opens (creating if needed) the mount-level lock file
// and acquires an exclusive POSIX flock on it. The returned release
// function unlocks the file and closes the descriptor; callers should
// `defer release()` immediately after a successful acquire.
//
// One mount-wide lock serializes all mutating operations
// (write/edit/rm/mv). This is the contract that makes CAS safe under
// concurrent sibling-process mutation: the version check and the on-disk
// mutation happen with no other writer interleaved.
func acquireMountLock(mount string) (release func(), err error) {
	if mount == "" {
		return nil, fmt.Errorf("mount is empty")
	}
	if err := os.MkdirAll(mount, 0o755); err != nil {
		return nil, fmt.Errorf("ensure mount dir: %w", err)
	}
	lockPath := filepath.Join(mount, lockFileName)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

package mycelium

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// syncDir fsyncs a directory after metadata changes such as rename, remove, or
// file creation. Some platforms/filesystems reject directory fsync with EINVAL;
// treat that as a best-effort no-op so tests and non-POSIX dev mounts remain usable.
func syncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		if errors.Is(err, syscall.EINVAL) {
			return nil
		}
		return err
	}
	return nil
}

// syncDirAncestors fsyncs dir and each parent up to and including stop. Use it
// after MkdirAll or a file create in a newly-created directory tree so parent
// directory entries are also durable, not only the leaf directory.
func syncDirAncestors(dir, stop string) error {
	current := filepath.Clean(dir)
	stop = filepath.Clean(stop)
	for {
		if err := syncDir(current); err != nil {
			return err
		}
		if current == stop {
			return nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

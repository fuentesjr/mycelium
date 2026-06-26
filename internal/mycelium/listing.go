package mycelium

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// isDotEntry reports whether the final path element starts with '.'.
func isDotEntry(name string) bool {
	return strings.HasPrefix(name, ".")
}

// listFiles walks mount and collects relative paths of files, skipping dotfiles
// and dot-directories. When pattern is non-empty, only matching files are
// included. If the pattern contains "/" the full relative path is matched;
// otherwise only the basename is matched. The returned slice is sorted
// alphabetically (forward-slash paths).
func listFiles(mount string, recursive bool, pattern string) ([]string, error) {
	usePattern := pattern != ""
	useFullPath := strings.Contains(pattern, "/")
	var results []string

	err := filepath.WalkDir(mount, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()

		// Skip the mount root itself.
		if absPath == filepath.Clean(mount) {
			return nil
		}

		// Skip dot entries (files and directories).
		if isDotEntry(name) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if !recursive {
				return filepath.SkipDir
			}
			return nil
		}

		// It's a regular file (or symlink to one — we include it).
		rel, err := filepath.Rel(mount, absPath)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		if usePattern {
			subject := path.Base(relSlash)
			if useFullPath {
				subject = relSlash
			}
			matched, err := path.Match(pattern, subject)
			if err != nil {
				return err
			}
			if !matched {
				return nil
			}
		}
		results = append(results, relSlash)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(results)
	return results, nil
}

// listFilesAndPrint calls listFiles and writes results to out, one per line.
// It returns an exit code suitable for returning from a subcommand handler.
func listFilesAndPrint(out, errOut io.Writer, mount string, recursive bool, pattern string) int {
	if mount == "" {
		fmt.Fprintln(errOut, "mycelium ls: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}
	files, err := listFiles(mount, recursive, pattern)
	if err != nil {
		if err == path.ErrBadPattern {
			fmt.Fprintf(errOut, "mycelium ls: invalid pattern: %s\n", pattern)
			return ExitUsage
		}
		fmt.Fprintf(errOut, "mycelium ls: %v\n", err)
		return ExitGenericError
	}
	for _, f := range files {
		fmt.Fprintln(out, f)
	}
	return ExitOK
}

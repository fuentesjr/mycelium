package main

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
// and dot-directories.  When recursive is false only top-level files are
// included.  The returned slice is sorted alphabetically (forward-slash paths).
func listFiles(mount string, recursive bool) ([]string, error) {
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
		results = append(results, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(results)
	return results, nil
}

// globMatches walks mount recursively and returns paths (relative, forward-slash)
// that match pattern.  If pattern contains '/' the full relative path is matched;
// otherwise only the basename is matched.  Dotfiles and dot-directories are
// skipped.  The returned slice is sorted alphabetically.
func globMatches(mount, pattern string) ([]string, error) {
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

		// Skip dot entries.
		if isDotEntry(name) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(mount, absPath)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		var subject string
		if useFullPath {
			subject = relSlash
		} else {
			subject = path.Base(relSlash)
		}

		matched, err := path.Match(pattern, subject)
		if err != nil {
			// path.ErrBadPattern — propagate so the caller can distinguish it.
			return err
		}
		if matched {
			results = append(results, relSlash)
		}
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
func listFilesAndPrint(out, errOut io.Writer, mount string, recursive bool) int {
	if mount == "" {
		fmt.Fprintln(errOut, "mycelium ls: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}
	files, err := listFiles(mount, recursive)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium ls: %v\n", err)
		return ExitGenericError
	}
	for _, f := range files {
		fmt.Fprintln(out, f)
	}
	return ExitOK
}

// globAndPrint calls globMatches and writes results to out, one per line.
// It returns an exit code suitable for returning from a subcommand handler.
func globAndPrint(out, errOut io.Writer, mount, pattern string) int {
	if mount == "" {
		fmt.Fprintln(errOut, "mycelium glob: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}
	files, err := globMatches(mount, pattern)
	if err != nil {
		// Check whether the pattern itself was invalid.
		if err == path.ErrBadPattern {
			fmt.Fprintf(errOut, "mycelium glob: invalid pattern: %s\n", pattern)
			return ExitUsage
		}
		fmt.Fprintf(errOut, "mycelium glob: %v\n", err)
		return ExitGenericError
	}
	for _, f := range files {
		fmt.Fprintln(out, f)
	}
	return ExitOK
}


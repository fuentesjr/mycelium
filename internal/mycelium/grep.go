package mycelium

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepOptions holds the validated inputs for a grep operation.
type GrepOptions struct {
	Pattern   string
	PathScope string // relative path under mount; "" means mount root
	Regex     bool
	Format    string // "text" or "json"
	Limit     int
}

// grepMatch is a single line match result.
type grepMatch struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

// grepFiles walks the filesystem under mount (scoped to opts.PathScope if set),
// searches for opts.Pattern in each file, and writes results to out.
// Per-file read errors are logged to errOut as warnings and do not abort the walk.
// Returns an exit code.
func grepFiles(out, errOut io.Writer, mount string, opts GrepOptions) int {
	// Validate mount.
	if mount == "" {
		fmt.Fprintln(errOut, "mycelium grep: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	// Determine the root to walk.
	var walkRoot string
	if opts.PathScope == "" {
		walkRoot = filepath.Clean(mount)
	} else {
		abs, err := resolveUnderMount(mount, opts.PathScope)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium grep: %v\n", err)
			if errors.Is(err, ErrMountUnset) {
				return ExitGenericError
			}
			return ExitUsage
		}
		if err := rejectSymlinkComponents(mount, abs); err != nil {
			fmt.Fprintf(errOut, "mycelium grep: %v\n", err)
			return ExitGenericError
		}
		// Verify the resolved path exists.
		if _, err := os.Stat(abs); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(errOut, "mycelium grep: %s: not found\n", opts.PathScope)
				return ExitGenericError
			}
			fmt.Fprintf(errOut, "mycelium grep: %v\n", err)
			return ExitGenericError
		}
		walkRoot = abs
	}

	// Compile regex once up front if needed.
	var re *regexp.Regexp
	if opts.Regex {
		var err error
		re, err = regexp.Compile(opts.Pattern)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium grep: invalid regex: %v\n", err)
			return ExitUsage
		}
	}

	cleanMount := filepath.Clean(mount)
	var matches []grepMatch
	truncated := false

	err := filepath.WalkDir(walkRoot, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()

		// Skip the walk root itself only when it is a directory; a file scope is searchable.
		if absPath == walkRoot && d.IsDir() {
			return nil
		}

		// Skip dot entries (files and directories).
		if isDotEntry(name) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories (we only process files).
		if d.IsDir() {
			return nil
		}

		// Compute relative path from mount root (not walkRoot) for output.
		rel, err := filepath.Rel(cleanMount, absPath)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		// Scan for one extra match so exact-limit results are not marked truncated.
		fileMatches, scanErr := scanFile(absPath, relSlash, opts.Pattern, re, opts.Limit+1-len(matches))
		if scanErr != nil {
			// Per-file errors are warnings, not fatal.
			fmt.Fprintf(errOut, "mycelium grep: skip %s: %v\n", relSlash, scanErr)
			return nil
		}

		matches = append(matches, fileMatches...)

		if len(matches) > opts.Limit {
			truncated = true
			matches = matches[:opts.Limit]
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipAll) {
		fmt.Fprintf(errOut, "mycelium grep: %v\n", err)
		return ExitGenericError
	}

	switch opts.Format {
	case "text":
		for _, m := range matches {
			fmt.Fprintf(out, "%s:%d:%s\n", m.Path, m.Line, m.Text)
		}
		if truncated {
			fmt.Fprintf(out, "(truncated: limit reached at %d matches)\n", len(matches))
		}
	case "json":
		result := struct {
			Matches   []grepMatch `json:"matches"`
			Truncated bool        `json:"truncated"`
		}{
			Matches:   matches,
			Truncated: truncated,
		}
		if result.Matches == nil {
			result.Matches = []grepMatch{}
		}
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(errOut, "mycelium grep: encode json: %v\n", err)
			return ExitGenericError
		}
	}

	return ExitOK
}

// scanFile reads absPath line by line and returns matches up to quota lines.
// relSlash is the forward-slash relative path used in match records.
func scanFile(absPath, relSlash, pattern string, re *regexp.Regexp, quota int) (matches []grepMatch, err error) {
	if quota <= 0 {
		return nil, nil
	}

	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Use default MaxScanTokenSize (64KB). If a line exceeds it, skip the file.
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		var matched bool
		if re != nil {
			matched = re.MatchString(line)
		} else {
			matched = strings.Contains(line, pattern)
		}

		if !matched {
			continue
		}

		matches = append(matches, grepMatch{
			Path: relSlash,
			Line: lineNum,
			Text: line,
		})

		if len(matches) >= quota {
			return matches, nil
		}
	}

	if err := scanner.Err(); err != nil {
		// If the error is a buffer overflow (line too long), skip the file silently.
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, nil
		}
		return nil, err
	}

	return matches, nil
}

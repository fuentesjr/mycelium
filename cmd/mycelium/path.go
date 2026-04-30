package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	ErrMountUnset      = errors.New("MYCELIUM_MOUNT is not set")
	ErrPathEmpty       = errors.New("path is empty")
	ErrPathAbsolute    = errors.New("path must be relative to mount")
	ErrPathEscapesRoot = errors.New("path escapes mount root")
	ErrReservedPath    = errors.New("path uses reserved '_' prefix")
)

func resolveUnderMount(mount, requested string) (string, error) {
	if mount == "" {
		return "", ErrMountUnset
	}
	if requested == "" {
		return "", ErrPathEmpty
	}
	if filepath.IsAbs(requested) {
		return "", ErrPathAbsolute
	}
	cleanedMount := filepath.Clean(mount)
	joined := filepath.Join(cleanedMount, requested)
	cleaned := filepath.Clean(joined)
	rel, err := filepath.Rel(cleanedMount, cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathEscapesRoot
	}
	return cleaned, nil
}

// relForwardSlash converts an absolute path to a forward-slash relative path
// from the mount root. Used to produce the relative log paths stored in entries.
func relForwardSlash(mount, abs string) string {
	rel, err := filepath.Rel(mount, abs)
	if err != nil {
		return abs
	}
	return strings.ReplaceAll(rel, string(filepath.Separator), "/")
}

// resolveAgentWritable resolves requested under mount and rejects any path
// whose first segment starts with '_'. Used by all agent-facing write paths.
// Internal binary writes (auto-log, mycelium log routing) use resolveUnderMount.
func resolveAgentWritable(mount, requested string) (string, error) {
	abs, err := resolveUnderMount(mount, requested)
	if err != nil {
		return "", err
	}
	// Check whether the first path segment of the cleaned requested starts with '_'.
	// We use the cleaned relative form: filepath.Clean strips leading ./ etc.
	cleaned := filepath.Clean(requested)
	// filepath.Clean on a relative path starting with _ keeps the _ as first char.
	// Split on separator to get the first segment.
	firstSeg := cleaned
	if idx := strings.IndexRune(cleaned, filepath.Separator); idx >= 0 {
		firstSeg = cleaned[:idx]
	}
	if strings.HasPrefix(firstSeg, "_") {
		return "", ErrReservedPath
	}
	return abs, nil
}

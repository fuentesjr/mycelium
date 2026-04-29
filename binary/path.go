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

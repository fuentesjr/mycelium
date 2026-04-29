package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// editFile reads the file at requested (resolved under mount), replaces exactly
// one occurrence of oldStr with newStr, writes the result atomically, and
// returns the new version string.
func editFile(errOut io.Writer, mount, requested, oldStr, newStr, expectedVersion string) (version string, rc int) {
	abs, err := resolveUnderMount(mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium edit: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(errOut, "mycelium edit: %s: not found\n", requested)
			return "", ExitGenericError
		}
		fmt.Fprintf(errOut, "mycelium edit: %v\n", err)
		return "", ExitGenericError
	}
	content := string(data)

	count := strings.Count(content, oldStr)
	switch {
	case count == 0:
		fmt.Fprintf(errOut, "mycelium edit: old string not found in %s\n", requested)
		return "", ExitGenericError
	case count > 1:
		fmt.Fprintf(errOut, "mycelium edit: old string is ambiguous: %d matches in %s\n", count, requested)
		return "", ExitGenericError
	}

	if expectedVersion != "" {
		if rc := checkExpectedVersion(errOut, "edit", abs, expectedVersion); rc != ExitOK {
			return "", rc
		}
	}

	newContent := strings.Replace(content, oldStr, newStr, 1)
	if err := atomicWrite(abs, []byte(newContent)); err != nil {
		fmt.Fprintf(errOut, "mycelium edit: %v\n", err)
		return "", ExitGenericError
	}

	sum := sha256.Sum256([]byte(newContent))
	ver := versionPrefix + hex.EncodeToString(sum[:])
	return ver, ExitOK
}

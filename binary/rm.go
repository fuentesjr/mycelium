package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// removeFile resolves requested under mount, checks CAS if expectedVersion is
// non-empty, captures the prior sha256, deletes the file, and returns the
// prior version string. On failure it writes a diagnostic to errOut and
// returns a non-zero exit code.
func removeFile(errOut io.Writer, mount, requested, expectedVersion string) (priorVersion string, rc int) {
	abs, err := resolveUnderMount(mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium rm: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}

	// Capture the current version (sha256) before touching anything.
	ver, err := currentVersion(abs)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium rm: %v\n", err)
		return "", ExitGenericError
	}
	// currentVersion returns "sha256:absent" when the file doesn't exist.
	if ver == versionPrefix+"absent" {
		fmt.Fprintf(errOut, "mycelium rm: %s: not found\n", requested)
		return "", ExitGenericError
	}

	if expectedVersion != "" {
		if rc := checkExpectedVersion(errOut, "rm", abs, expectedVersion); rc != ExitOK {
			return "", rc
		}
	}

	if err := os.Remove(abs); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(errOut, "mycelium rm: %s: not found\n", requested)
			return "", ExitGenericError
		}
		fmt.Fprintf(errOut, "mycelium rm: %v\n", err)
		return "", ExitGenericError
	}

	return ver, ExitOK
}

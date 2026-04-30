package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"unicode/utf8"
)

// moveFile resolves src and dst under mount, validates preconditions, checks
// CAS if expectedVersion is non-empty, then renames src to dst atomically.
// It returns the sha256 version of the moved content and a zero exit code on
// success, or a diagnostic on errOut and a non-zero exit code on failure.
func moveFile(errOut io.Writer, mount, src, dst, expectedVersion string, includeContent bool) (version string, rc int) {
	srcAbs, err := resolveUnderMount(mount, src)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}

	dstAbs, err := resolveUnderMount(mount, dst)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}

	if srcAbs == dstAbs {
		fmt.Fprintf(errOut, "mycelium mv: src and dst are the same\n")
		return "", ExitGenericError
	}

	release, err := acquireMountLock(mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		return "", ExitGenericError
	}
	defer release()

	// Check that src exists and capture its version.
	ver, err := currentVersion(srcAbs)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		return "", ExitGenericError
	}
	if ver == versionPrefix+"absent" {
		fmt.Fprintf(errOut, "mycelium mv: %s: not found\n", src)
		return "", ExitGenericError
	}

	// Refuse to overwrite an existing dst.
	if _, err := os.Stat(dstAbs); err == nil {
		dstRel := relForwardSlash(mount, dstAbs)
		dstVer, verErr := currentVersion(dstAbs)
		if verErr != nil {
			fmt.Fprintf(errOut, "mycelium mv: %v\n", verErr)
			return "", ExitGenericError
		}
		env := conflictEnvelope{
			Error:          "destination_exists",
			Op:             "mv",
			Path:           dstRel,
			CurrentVersion: dstVer,
		}
		if includeContent {
			fileBytes, readErr := os.ReadFile(dstAbs)
			if readErr == nil && utf8.Valid(fileBytes) {
				env.CurrentContent = string(fileBytes)
			}
		}
		line, _ := json.Marshal(env)
		line = append(line, '\n')
		_, _ = errOut.Write(line)
		return "", ExitConflict
	} else if !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(errOut, "mycelium mv: stat dst: %v\n", err)
		return "", ExitGenericError
	}

	if expectedVersion != "" {
		srcRel := relForwardSlash(mount, srcAbs)
		if rc := checkExpectedVersion(errOut, "mv", srcRel, srcAbs, expectedVersion, includeContent); rc != ExitOK {
			return "", rc
		}
	}

	if err := os.MkdirAll(filepath.Dir(dstAbs), 0o755); err != nil {
		fmt.Fprintf(errOut, "mycelium mv: mkdir: %v\n", err)
		return "", ExitGenericError
	}

	if err := os.Rename(srcAbs, dstAbs); err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		return "", ExitGenericError
	}

	return ver, ExitOK
}

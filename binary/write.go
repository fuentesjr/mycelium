package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const versionPrefix = "sha256:"

func writeFile(in io.Reader, errOut io.Writer, mount, requested, expectedVersion string, includeContent bool) (version string, rc int) {
	abs, err := resolveUnderMount(mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}
	content, err := io.ReadAll(in)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: read stdin: %v\n", err)
		return "", ExitGenericError
	}
	release, err := acquireMountLock(mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		return "", ExitGenericError
	}
	defer release()
	if expectedVersion != "" {
		mountRel := relForwardSlash(mount, abs)
		if rc := checkExpectedVersion(errOut, "write", mountRel, abs, expectedVersion, includeContent); rc != ExitOK {
			return "", rc
		}
	}
	if err := atomicWrite(abs, content); err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		return "", ExitGenericError
	}
	sum := sha256.Sum256(content)
	ver := versionPrefix + hex.EncodeToString(sum[:])
	return ver, ExitOK
}

// conflictEnvelope is the JSON structure emitted to stderr on a CAS conflict.
type conflictEnvelope struct {
	Error           string `json:"error"`
	Op              string `json:"op"`
	Path            string `json:"path"`
	CurrentVersion  string `json:"current_version"`
	ExpectedVersion string `json:"expected_version,omitempty"`
	CurrentContent  string `json:"current_content,omitempty"`
}

// checkExpectedVersion checks whether expected matches the on-disk version of
// abs. mountRel is the forward-slash path relative to mount (for the envelope).
// If includeContent is true and the file exists and contains valid UTF-8, the
// envelope will include a current_content field.
func checkExpectedVersion(errOut io.Writer, op, mountRel, abs, expected string, includeContent bool) int {
	if !strings.HasPrefix(expected, versionPrefix) {
		fmt.Fprintf(errOut, "mycelium %s: expected-version must start with %q\n", op, versionPrefix)
		return ExitUsage
	}
	current, err := currentVersion(abs)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium %s: %v\n", op, err)
		return ExitGenericError
	}
	if current != expected {
		env := conflictEnvelope{
			Error:           "conflict",
			Op:              op,
			Path:            mountRel,
			CurrentVersion:  current,
			ExpectedVersion: expected,
		}
		if includeContent && current != versionPrefix+"absent" {
			fileBytes, readErr := os.ReadFile(abs)
			if readErr == nil && utf8.Valid(fileBytes) {
				env.CurrentContent = string(fileBytes)
			}
		}
		line, _ := json.Marshal(env)
		line = append(line, '\n')
		_, _ = errOut.Write(line)
		return ExitConflict
	}
	return ExitOK
}

func currentVersion(abs string) (string, error) {
	f, err := os.Open(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return versionPrefix + "absent", nil
		}
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return versionPrefix + hex.EncodeToString(h.Sum(nil)), nil
}

func atomicWrite(abs string, content []byte) error {
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".mycelium-write-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpPath, abs); err != nil {
		cleanup()
		return err
	}
	return nil
}

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const versionPrefix = "sha256:"

func writeFile(in io.Reader, out, errOut io.Writer, mount, requested, expectedVersion string) int {
	abs, err := resolveUnderMount(mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return ExitGenericError
		}
		return ExitUsage
	}
	content, err := io.ReadAll(in)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: read stdin: %v\n", err)
		return ExitGenericError
	}
	if expectedVersion != "" {
		if rc := checkExpectedVersion(errOut, abs, expectedVersion); rc != ExitOK {
			return rc
		}
	}
	if err := atomicWrite(abs, content); err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		return ExitGenericError
	}
	sum := sha256.Sum256(content)
	version := versionPrefix + hex.EncodeToString(sum[:])
	fmt.Fprintf(out, `{"version":%q,"log_status":"ok"}`+"\n", version)
	return ExitOK
}

func checkExpectedVersion(errOut io.Writer, abs, expected string) int {
	if !strings.HasPrefix(expected, versionPrefix) {
		fmt.Fprintf(errOut, "mycelium write: expected-version must start with %q\n", versionPrefix)
		return ExitUsage
	}
	current, err := currentVersion(abs)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		return ExitGenericError
	}
	if current != expected {
		fmt.Fprintf(errOut, "mycelium write: version conflict: have %s, expected %s\n", current, expected)
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

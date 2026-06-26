package mycelium

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
)

const versionPrefix = "sha256:"

func hashVersion(content []byte) string {
	sum := sha256.Sum256(content)
	return versionPrefix + hex.EncodeToString(sum[:])
}

// conflictEnvelope is the JSON structure emitted to stderr on a CAS conflict.
type conflictEnvelope struct {
	Error           string `json:"error"`
	Op              string `json:"op"`
	Path            string `json:"path"`
	CurrentVersion  string `json:"current_version"`
	ExpectedVersion string `json:"expected_version,omitempty"`
	Rationale       string `json:"rationale,omitempty"`
}

// checkExpectedVersion checks whether expected matches the on-disk version of
// abs. mountRel is the forward-slash path relative to mount (for the envelope).
// rationale, when non-empty, is propagated to the conflict envelope.
func checkExpectedVersion(errOut io.Writer, op, mountRel, abs, expected string, rationale string) int {
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
			Rationale:       rationale,
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

func atomicWrite(abs string, content []byte, syncStops ...string) error {
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
	if err := tmp.Sync(); err != nil {
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
	if len(syncStops) > 0 && syncStops[0] != "" {
		if err := syncDirAncestors(dir, syncStops[0]); err != nil {
			return err
		}
	} else if err := syncDir(dir); err != nil {
		return err
	}
	return nil
}

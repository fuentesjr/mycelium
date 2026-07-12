package mycelium

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func mutatingWrite(errOut io.Writer, id Identity, requested string, content []byte, expectedVersion string, rationale string) (version string, rc int) {
	abs, err := resolveUnderMount(id.Mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}

	release, err := acquireMountLock(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		return "", ExitGenericError
	}
	defer release()

	if rc := blockLegacyPendingTransactions(errOut, id.Mount); rc != ExitOK {
		return "", rc
	}
	if err := rejectSymlinkComponents(id.Mount, abs); err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		return "", ExitGenericError
	}

	prior, err := currentVersion(abs)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium write: %v\n", err)
		return "", ExitGenericError
	}
	if expectedVersion != "" {
		mountRel := relForwardSlash(id.Mount, abs)
		if rc := checkExpectedVersion(errOut, "write", mountRel, abs, expectedVersion, rationale); rc != ExitOK {
			return "", rc
		}
	}

	version = hashVersion(content)
	entry, rc := newMutationLogEntry(errOut, id, "write", LogEntry{
		Op:           "write",
		Path:         requested,
		PriorVersion: prior,
		Version:      version,
		Rationale:    rationale,
	})
	if rc != ExitOK {
		return "", rc
	}

	rc = commitMutation(errOut, id, "write", entry, func() error {
		return atomicWrite(abs, content, id.Mount)
	})
	if rc != ExitOK {
		return "", rc
	}
	return version, ExitOK
}

func mutatingEdit(errOut io.Writer, id Identity, requested, oldStr, newStr, expectedVersion string, rationale string) (version string, rc int) {
	abs, err := resolveUnderMount(id.Mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium edit: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}

	release, err := acquireMountLock(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium edit: %v\n", err)
		return "", ExitGenericError
	}
	defer release()

	if rc := blockLegacyPendingTransactions(errOut, id.Mount); rc != ExitOK {
		return "", rc
	}
	if err := rejectSymlinkComponents(id.Mount, abs); err != nil {
		fmt.Fprintf(errOut, "mycelium edit: %v\n", err)
		return "", ExitGenericError
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

	prior := hashVersion(data)
	if expectedVersion != "" {
		mountRel := relForwardSlash(id.Mount, abs)
		if rc := checkExpectedVersion(errOut, "edit", mountRel, abs, expectedVersion, rationale); rc != ExitOK {
			return "", rc
		}
	}

	newContent := strings.Replace(content, oldStr, newStr, 1)
	newBytes := []byte(newContent)
	version = hashVersion(newBytes)
	entry, rc := newMutationLogEntry(errOut, id, "edit", LogEntry{
		Op:           "edit",
		Path:         requested,
		PriorVersion: prior,
		Version:      version,
		Rationale:    rationale,
	})
	if rc != ExitOK {
		return "", rc
	}

	rc = commitMutation(errOut, id, "edit", entry, func() error {
		return atomicWrite(abs, newBytes, id.Mount)
	})
	if rc != ExitOK {
		return "", rc
	}
	return version, ExitOK
}

func mutatingRemove(errOut io.Writer, id Identity, requested, expectedVersion string, rationale string) (priorVersion string, rc int) {
	abs, err := resolveUnderMount(id.Mount, requested)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium rm: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}

	release, err := acquireMountLock(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium rm: %v\n", err)
		return "", ExitGenericError
	}
	defer release()

	if rc := blockLegacyPendingTransactions(errOut, id.Mount); rc != ExitOK {
		return "", rc
	}
	if err := rejectSymlinkComponents(id.Mount, abs); err != nil {
		fmt.Fprintf(errOut, "mycelium rm: %v\n", err)
		return "", ExitGenericError
	}

	priorVersion, err = currentVersion(abs)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium rm: %v\n", err)
		return "", ExitGenericError
	}
	if priorVersion == versionPrefix+"absent" {
		fmt.Fprintf(errOut, "mycelium rm: %s: not found\n", requested)
		return "", ExitGenericError
	}

	if expectedVersion != "" {
		mountRel := relForwardSlash(id.Mount, abs)
		if rc := checkExpectedVersion(errOut, "rm", mountRel, abs, expectedVersion, rationale); rc != ExitOK {
			return "", rc
		}
	}

	entry, rc := newMutationLogEntry(errOut, id, "rm", LogEntry{
		Op:           "rm",
		Path:         requested,
		PriorVersion: priorVersion,
		Rationale:    rationale,
	})
	if rc != ExitOK {
		return "", rc
	}

	rc = commitMutation(errOut, id, "rm", entry, func() error {
		if err := os.Remove(abs); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("%s: not found", requested)
			}
			return err
		}
		return syncDirAncestors(filepath.Dir(abs), id.Mount)
	})
	if rc != ExitOK {
		return "", rc
	}
	return priorVersion, ExitOK
}

func mutatingMove(errOut io.Writer, id Identity, src, dst, expectedVersion string, rationale string) (version string, rc int) {
	srcAbs, err := resolveUnderMount(id.Mount, src)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		if errors.Is(err, ErrMountUnset) {
			return "", ExitGenericError
		}
		return "", ExitUsage
	}
	dstAbs, err := resolveUnderMount(id.Mount, dst)
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

	release, err := acquireMountLock(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		return "", ExitGenericError
	}
	defer release()

	if rc := blockLegacyPendingTransactions(errOut, id.Mount); rc != ExitOK {
		return "", rc
	}
	if err := rejectSymlinkComponents(id.Mount, srcAbs); err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		return "", ExitGenericError
	}
	if err := rejectSymlinkComponents(id.Mount, dstAbs); err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		return "", ExitGenericError
	}

	version, err = currentVersion(srcAbs)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium mv: %v\n", err)
		return "", ExitGenericError
	}
	if version == versionPrefix+"absent" {
		fmt.Fprintf(errOut, "mycelium mv: %s: not found\n", src)
		return "", ExitGenericError
	}

	if _, err := os.Stat(dstAbs); err == nil {
		if rc := emitDestinationExists(errOut, id.Mount, dstAbs, rationale); rc != ExitOK {
			return "", rc
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(errOut, "mycelium mv: stat dst: %v\n", err)
		return "", ExitGenericError
	}

	if expectedVersion != "" {
		srcRel := relForwardSlash(id.Mount, srcAbs)
		if rc := checkExpectedVersion(errOut, "mv", srcRel, srcAbs, expectedVersion, rationale); rc != ExitOK {
			return "", rc
		}
	}

	entry, rc := newMutationLogEntry(errOut, id, "mv", LogEntry{
		Op:        "mv",
		From:      src,
		Path:      dst,
		Version:   version,
		Rationale: rationale,
	})
	if rc != ExitOK {
		return "", rc
	}

	rc = commitMutation(errOut, id, "mv", entry, func() error {
		dstDir := filepath.Dir(dstAbs)
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		if err := rejectSymlinkComponents(id.Mount, dstAbs); err != nil {
			return err
		}
		if err := os.Rename(srcAbs, dstAbs); err != nil {
			return err
		}
		if err := syncDirAncestors(filepath.Dir(srcAbs), id.Mount); err != nil {
			return err
		}
		if dstDir != filepath.Dir(srcAbs) {
			if err := syncDirAncestors(dstDir, id.Mount); err != nil {
				return err
			}
		}
		return nil
	})
	if rc != ExitOK {
		return "", rc
	}
	return version, ExitOK
}

func newMutationLogEntry(errOut io.Writer, id Identity, op string, entry LogEntry) (LogEntry, int) {
	now := time.Now()
	txID, err := newTxID(now)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium %s: generate tx id: %v\n", op, err)
		return LogEntry{}, ExitGenericError
	}
	entry.TxID = txID
	return completeLogEntry(id, entry, now), ExitOK
}

func commitMutation(errOut io.Writer, id Identity, op string, entry LogEntry, apply func() error) int {
	if err := apply(); err != nil {
		fmt.Fprintf(errOut, "mycelium %s: %v\n", op, err)
		return ExitGenericError
	}
	if err := appendActivityEntryDurable(id.Mount, entry); err != nil {
		fmt.Fprintf(errOut, "mycelium %s: log entry write failed after content commit: %v\n", op, err)
		return ExitGenericError
	}
	return ExitOK
}

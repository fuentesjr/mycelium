package mycelium

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const pendingTxDirName = "pending"

type pendingTransaction struct {
	TxID         string   `json:"tx_id"`
	Op           string   `json:"op"`
	Path         string   `json:"path,omitempty"`
	From         string   `json:"from,omitempty"`
	PriorVersion string   `json:"prior_version,omitempty"`
	Version      string   `json:"version,omitempty"`
	Activity     LogEntry `json:"activity"`
}

func pendingTxDir(mount string) string {
	return filepath.Join(mount, "_tx", pendingTxDirName)
}

func pendingTxPath(mount, txID string) string {
	return filepath.Join(pendingTxDir(mount), txID+".json")
}

func newContentTransaction(id Identity, txID string, now time.Time, activity LogEntry, priorVersion, postVersion string) pendingTransaction {
	activity.TxID = txID
	activity = completeLogEntry(id, activity, now)
	return pendingTransaction{
		TxID:         txID,
		Op:           activity.Op,
		Path:         activity.Path,
		From:         activity.From,
		PriorVersion: priorVersion,
		Version:      postVersion,
		Activity:     activity,
	}
}

func preparePendingTransaction(mount string, tx pendingTransaction) error {
	if tx.TxID == "" {
		return fmt.Errorf("transaction id is empty")
	}
	dir := pendingTxDir(mount)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir pending tx dir: %w", err)
	}
	// Durably record the full _tx/pending directory chain before publishing the transaction file.
	if err := syncDirAncestors(dir, mount); err != nil {
		return fmt.Errorf("sync pending tx dirs: %w", err)
	}
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pending tx: %w", err)
	}
	data = append(data, '\n')
	if err := atomicWrite(pendingTxPath(mount, tx.TxID), data, mount); err != nil {
		return fmt.Errorf("write pending tx: %w", err)
	}
	return nil
}

func removePendingTransaction(mount, txID string) error {
	path := pendingTxPath(mount, txID)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return err
	}
	return nil
}

func commitContentTransaction(errOut io.Writer, id Identity, tx pendingTransaction, apply func() error) int {
	if err := preparePendingTransaction(id.Mount, tx); err != nil {
		fmt.Fprintf(errOut, "mycelium %s: %v\n", tx.Op, err)
		return ExitGenericError
	}

	if err := apply(); err != nil {
		// If the atomic content operation failed before committing, remove the
		// prepared transaction so a later recovery does not need to reason about it.
		_ = removePendingTransaction(id.Mount, tx.TxID)
		fmt.Fprintf(errOut, "mycelium %s: %v\n", tx.Op, err)
		return ExitGenericError
	}

	if err := appendActivityEntryDurable(id.Mount, tx.Activity); err != nil {
		// Content may already be committed. Leave _tx/pending in place so recovery
		// can make the missing activity entry durable before any later mutation.
		fmt.Fprintf(errOut, "mycelium %s: log entry write failed: %v\n", tx.Op, err)
		return ExitGenericError
	}

	if err := removePendingTransaction(id.Mount, tx.TxID); err != nil {
		fmt.Fprintf(errOut, "mycelium %s: remove pending tx: %v\n", tx.Op, err)
		return ExitGenericError
	}
	return ExitOK
}

func recoverPendingTransactions(errOut io.Writer, id Identity) int {
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	dir := pendingTxDir(id.Mount)
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		fmt.Fprintf(errOut, "mycelium: scan pending transactions: %v\n", err)
		return ExitGenericError
	}
	if len(matches) == 0 {
		return ExitOK
	}
	sort.Strings(matches)

	for _, path := range matches {
		if rc := recoverPendingTransaction(errOut, id, path); rc != ExitOK {
			return rc
		}
	}
	return ExitOK
}

func recoverPendingTransaction(errOut io.Writer, id Identity, path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ExitOK
		}
		fmt.Fprintf(errOut, "mycelium: read pending transaction %s: %v\n", relForwardSlash(id.Mount, path), err)
		return ExitGenericError
	}
	var tx pendingTransaction
	if err := json.Unmarshal(data, &tx); err != nil {
		fmt.Fprintf(errOut, "mycelium: unrecoverable pending transaction %s: invalid JSON: %v\n", relForwardSlash(id.Mount, path), err)
		return ExitGenericError
	}
	if err := normalizePendingTransaction(&tx); err != nil {
		fmt.Fprintf(errOut, "mycelium: unrecoverable pending transaction %s: %v\n", relForwardSlash(id.Mount, path), err)
		return ExitGenericError
	}

	logged, err := activityLogHasTxID(id.Mount, tx.TxID)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium: scan activity for pending transaction %s: %v\n", tx.TxID, err)
		return ExitGenericError
	}
	if logged {
		if err := removePendingTransaction(id.Mount, tx.TxID); err != nil {
			fmt.Fprintf(errOut, "mycelium: remove recovered pending transaction %s: %v\n", tx.TxID, err)
			return ExitGenericError
		}
		return ExitOK
	}

	state, err := pendingTransactionState(id.Mount, tx)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium: recover pending transaction %s: %v\n", tx.TxID, err)
		return ExitGenericError
	}

	switch state {
	case "pre":
		if err := removePendingTransaction(id.Mount, tx.TxID); err != nil {
			fmt.Fprintf(errOut, "mycelium: remove uncommitted pending transaction %s: %v\n", tx.TxID, err)
			return ExitGenericError
		}
		return ExitOK
	case "post":
		recovered := tx.Activity
		recovered.Recovered = true
		if err := appendActivityEntryDurable(id.Mount, recovered); err != nil {
			fmt.Fprintf(errOut, "mycelium: recover pending transaction %s: append activity: %v\n", tx.TxID, err)
			return ExitGenericError
		}
		if err := removePendingTransaction(id.Mount, tx.TxID); err != nil {
			fmt.Fprintf(errOut, "mycelium: remove recovered pending transaction %s: %v\n", tx.TxID, err)
			return ExitGenericError
		}
		return ExitOK
	default:
		fmt.Fprintf(errOut, "mycelium: unrecoverable pending transaction %s at %s: content matches neither precondition nor postcondition\n", tx.TxID, relForwardSlash(id.Mount, path))
		return ExitGenericError
	}
}

func normalizePendingTransaction(tx *pendingTransaction) error {
	if tx.TxID == "" {
		return fmt.Errorf("tx_id is required")
	}
	if tx.Op == "" {
		tx.Op = tx.Activity.Op
	}
	switch tx.Op {
	case "write", "edit", "rm", "mv":
	default:
		return fmt.Errorf("unsupported op %q", tx.Op)
	}
	if tx.Path == "" {
		tx.Path = tx.Activity.Path
	}
	if tx.From == "" {
		tx.From = tx.Activity.From
	}
	if tx.Activity.TxID == "" {
		tx.Activity.TxID = tx.TxID
	}
	if tx.Activity.TxID != tx.TxID {
		return fmt.Errorf("activity tx_id %q does not match tx_id %q", tx.Activity.TxID, tx.TxID)
	}
	if tx.Activity.Op == "" {
		tx.Activity.Op = tx.Op
	}
	if tx.Activity.Op != tx.Op {
		return fmt.Errorf("activity op %q does not match op %q", tx.Activity.Op, tx.Op)
	}
	if tx.Activity.Path == "" {
		tx.Activity.Path = tx.Path
	}
	if tx.Op != "mv" && tx.Path == "" {
		return fmt.Errorf("path is required")
	}
	if tx.Op == "mv" && (tx.From == "" || tx.Path == "") {
		return fmt.Errorf("mv requires from and path")
	}
	if tx.Activity.TS == "" {
		return fmt.Errorf("activity ts is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, tx.Activity.TS); err != nil {
		return fmt.Errorf("activity ts is invalid: %w", err)
	}
	if tx.Activity.AgentID == "" {
		// Keep the same fallback as activityLogPath/appendActivity.
		tx.Activity.AgentID = ""
	}
	return nil
}

func pendingTransactionState(mount string, tx pendingTransaction) (string, error) {
	switch tx.Op {
	case "write", "edit":
		abs, err := resolveUnderMount(mount, tx.Path)
		if err != nil {
			return "", err
		}
		cur, err := currentVersion(abs)
		if err != nil {
			return "", err
		}
		// Check postcondition first so no-op writes/edits (prior == version) are
		// recovered as committed if the process died after preparing the tx.
		if cur == tx.Version {
			return "post", nil
		}
		if cur == tx.PriorVersion {
			return "pre", nil
		}
		return "uncertain", nil
	case "rm":
		abs, err := resolveUnderMount(mount, tx.Path)
		if err != nil {
			return "", err
		}
		cur, err := currentVersion(abs)
		if err != nil {
			return "", err
		}
		if cur == versionPrefix+"absent" {
			return "post", nil
		}
		if cur == tx.PriorVersion {
			return "pre", nil
		}
		return "uncertain", nil
	case "mv":
		srcAbs, err := resolveUnderMount(mount, tx.From)
		if err != nil {
			return "", err
		}
		dstAbs, err := resolveUnderMount(mount, tx.Path)
		if err != nil {
			return "", err
		}
		srcCur, err := currentVersion(srcAbs)
		if err != nil {
			return "", err
		}
		dstCur, err := currentVersion(dstAbs)
		if err != nil {
			return "", err
		}
		movedVersion := tx.Version
		if movedVersion == "" {
			movedVersion = tx.PriorVersion
		}
		if srcCur == versionPrefix+"absent" && dstCur == movedVersion {
			return "post", nil
		}
		if srcCur == movedVersion && dstCur == versionPrefix+"absent" {
			return "pre", nil
		}
		return "uncertain", nil
	default:
		return "", fmt.Errorf("unsupported op %q", tx.Op)
	}
}

func activityLogHasTxID(mount, txID string) (bool, error) {
	if txID == "" {
		return false, nil
	}
	pattern := filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false, err
	}
	sort.Strings(matches)
	for _, path := range matches {
		found := false
		err := scanJSONLinesFile(path, func(line []byte) error {
			var partial struct {
				TxID string `json:"tx_id"`
			}
			if err := json.Unmarshal(line, &partial); err != nil {
				return nil
			}
			if partial.TxID == txID {
				found = true
			}
			return nil
		})
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

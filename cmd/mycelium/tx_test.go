package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type rawLogEntry map[string]any

func pendingTxFiles(t *testing.T, mount string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(mount, "_tx", "pending", "*.json"))
	if err != nil {
		t.Fatalf("glob pending tx: %v", err)
	}
	return matches
}

func writePendingTx(t *testing.T, mount string, tx pendingTransaction) {
	t.Helper()
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		t.Fatalf("marshal pending tx: %v", err)
	}
	dir := filepath.Join(mount, "_tx", "pending")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir pending tx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, tx.TxID+".json"), append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write pending tx: %v", err)
	}
}

func newTestPendingTx(t *testing.T, id Identity, op, path, from, prior, version string) pendingTransaction {
	t.Helper()
	txID := newULID()
	activity := LogEntry{
		Op:           op,
		Path:         path,
		From:         from,
		PriorVersion: prior,
		Version:      version,
	}
	return newContentTransaction(id, txID, fixedNow, activity, prior, version)
}

func TestSuccessfulContentMutationCleansPendingTx(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "hello\n", "write", "notes.md")
	if rc != ExitOK {
		t.Fatalf("write rc=%d stderr=%q", rc, errOut)
	}

	if got := pendingTxFiles(t, mount); len(got) != 0 {
		t.Fatalf("pending tx files after successful write: %v", got)
	}
	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("log entries: got %d, want 1", len(entries))
	}
	if entries[0].TxID == "" {
		t.Fatal("mutation log should include tx_id")
	}
}

func TestRecoveryRemovesUncommittedPendingTransaction(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	id := Identity{AgentID: "agent", SessionID: "sess", Mount: mount}

	content := []byte("before\n")
	writeTestFile(t, mount, "notes.md", string(content))
	prior := hashVersion(content)
	post := hashVersion([]byte("after\n"))
	tx := newTestPendingTx(t, id, "write", "notes.md", "", prior, post)
	writePendingTx(t, mount, tx)

	_, errOut, rc := runDispatchWithStdin(t, "new\n", "write", "other.md")
	if rc != ExitOK {
		t.Fatalf("write after recovery rc=%d stderr=%q", rc, errOut)
	}
	if got := pendingTxFiles(t, mount); len(got) != 0 {
		t.Fatalf("pending tx not removed: %v", got)
	}
	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("log entries: got %d, want only new write", len(entries))
	}
	if entries[0].TxID == tx.TxID {
		t.Fatalf("uncommitted tx should not have been logged: %+v", entries[0])
	}
}

func TestRecoveryLogsCommittedPendingTransaction(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "agent")
	t.Setenv("MYCELIUM_SESSION_ID", "sess")
	id := Identity{AgentID: "agent", SessionID: "sess", Mount: mount}

	prior := versionPrefix + "absent"
	content := []byte("committed\n")
	post := hashVersion(content)
	writeTestFile(t, mount, "notes.md", string(content))
	tx := newTestPendingTx(t, id, "write", "notes.md", "", prior, post)
	writePendingTx(t, mount, tx)

	_, errOut, rc := runDispatchWithStdin(t, "new\n", "write", "other.md")
	if rc != ExitOK {
		t.Fatalf("write after recovery rc=%d stderr=%q", rc, errOut)
	}
	if got := pendingTxFiles(t, mount); len(got) != 0 {
		t.Fatalf("pending tx not removed: %v", got)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 2 {
		t.Fatalf("log entries: got %d, want recovered + new write", len(entries))
	}
	if entries[0].TxID != tx.TxID {
		t.Fatalf("first entry tx_id: got %q, want %q", entries[0].TxID, tx.TxID)
	}
	if !entries[0].Recovered {
		t.Fatal("recovered entry should have recovered=true")
	}
	if entries[0].Version != post {
		t.Errorf("recovered version: got %q, want %q", entries[0].Version, post)
	}
}

func TestRecoveryRefusesUncertainPendingTransaction(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	id := Identity{AgentID: "agent", SessionID: "sess", Mount: mount}

	prior := hashVersion([]byte("before\n"))
	post := hashVersion([]byte("after\n"))
	writeTestFile(t, mount, "notes.md", "something-else\n")
	tx := newTestPendingTx(t, id, "write", "notes.md", "", prior, post)
	writePendingTx(t, mount, tx)

	_, errOut, rc := runDispatchWithStdin(t, "new\n", "write", "other.md")
	if rc != ExitGenericError {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "unrecoverable pending transaction") {
		t.Fatalf("stderr should mention unrecoverable pending transaction, got %q", errOut)
	}
	if got := pendingTxFiles(t, mount); len(got) != 1 {
		t.Fatalf("uncertain pending tx should remain, got %v", got)
	}
	if _, err := os.Stat(filepath.Join(mount, "other.md")); !os.IsNotExist(err) {
		t.Fatalf("new mutation should not proceed while recovery is uncertain, stat err=%v", err)
	}
}

func TestActivityLogEntryDurabilityPreservesPreparedTimestamp(t *testing.T) {
	mount := t.TempDir()
	when := fixedNow.Add(2 * time.Hour)
	entry := LogEntry{TS: when.Format(time.RFC3339Nano), AgentID: "agent", TxID: "tx", Op: "write", Path: "p.md", Version: "sha256:x"}
	if err := appendActivityEntryDurable(mount, entry); err != nil {
		t.Fatalf("appendActivityEntryDurable: %v", err)
	}
	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("entries: got %d, want 1", len(entries))
	}
	if entries[0].TS != entry.TS {
		t.Errorf("timestamp: got %q, want %q", entries[0].TS, entry.TS)
	}
}

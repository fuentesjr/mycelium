package mycelium

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeLegacyPendingTx(t *testing.T, mount, name string) string {
	t.Helper()
	dir := legacyPendingTxDir(mount)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir legacy tx dir: %v", err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, []byte(`{"tx_id":"`+name+`"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write legacy pending tx: %v", err)
	}
	return path
}

func assertGeneratedID(t *testing.T, got, prefix string) {
	t.Helper()
	parts := strings.Split(got, "-")
	if len(parts) != 3 {
		t.Fatalf("generated id %q should have three dash-separated parts", got)
	}
	if parts[0] != prefix {
		t.Fatalf("generated id prefix: got %q, want %q", parts[0], prefix)
	}
	if len(parts[1]) != 19 {
		t.Fatalf("generated id timestamp %q should be 19 zero-padded digits", parts[1])
	}
	if _, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
		t.Fatalf("generated id timestamp %q is not decimal: %v", parts[1], err)
	}
	if len(parts[2]) != 16 {
		t.Fatalf("generated id random suffix %q should be 16 hex chars", parts[2])
	}
	if _, err := hex.DecodeString(parts[2]); err != nil {
		t.Fatalf("generated id random suffix %q is not hex: %v", parts[2], err)
	}
}

func TestSuccessfulContentMutationDoesNotCreateTxJournal(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "hello\n", "write", "notes.md")
	if rc != ExitOK {
		t.Fatalf("write rc=%d stderr=%q", rc, errOut)
	}

	if _, err := os.Stat(filepath.Join(mount, "_tx")); !os.IsNotExist(err) {
		t.Fatalf("_tx should not be created after successful write, stat err=%v", err)
	}
	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("log entries: got %d, want 1", len(entries))
	}
	assertGeneratedID(t, entries[0].TxID, "tx")
}

func TestLegacyPendingTxBlocksWriteBeforeContentMutation(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	pendingPath := writeLegacyPendingTx(t, mount, "legacy-write")

	_, errOut, rc := runDispatchWithStdin(t, "new\n", "write", "notes.md")
	if rc != ExitGenericError {
		t.Fatalf("rc=%d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "legacy _tx/pending records found") {
		t.Fatalf("stderr should mention legacy pending records, got %q", errOut)
	}
	if !strings.Contains(errOut, "v0.2") {
		t.Fatalf("stderr should point to v0.2 recovery, got %q", errOut)
	}
	if !strings.Contains(errOut, relForwardSlash(mount, pendingPath)) {
		t.Fatalf("stderr should name first pending record, got %q", errOut)
	}
	if _, err := os.Stat(filepath.Join(mount, "notes.md")); !os.IsNotExist(err) {
		t.Fatalf("write should not proceed while legacy pending tx exists, stat err=%v", err)
	}
}

func TestLegacyPendingTxBlocksLogAppend(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeLegacyPendingTx(t, mount, "legacy-log")

	_, errOut, rc := runDispatch(t, "log", "context_signal", "--payload-json", "{}")
	if rc != ExitGenericError {
		t.Fatalf("rc=%d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "legacy _tx/pending records found") {
		t.Fatalf("stderr should mention legacy pending records, got %q", errOut)
	}
	if logExists(mount) {
		t.Fatal("log append should not proceed while legacy pending tx exists")
	}
}

func TestMutationAppendFailureFailsLoudAfterContentCommit(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	if err := os.WriteFile(filepath.Join(mount, "_activity"), []byte("not a dir\n"), 0o644); err != nil {
		t.Fatalf("seed _activity file: %v", err)
	}

	_, errOut, rc := runDispatchWithStdin(t, "committed\n", "write", "notes.md")
	if rc != ExitGenericError {
		t.Fatalf("rc=%d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "log entry write failed after content commit") {
		t.Fatalf("stderr should report post-commit log failure, got %q", errOut)
	}
	data, err := os.ReadFile(filepath.Join(mount, "notes.md"))
	if err != nil {
		t.Fatalf("content should remain committed after append failure: %v", err)
	}
	if string(data) != "committed\n" {
		t.Fatalf("content: got %q, want committed", data)
	}
	if _, err := os.Stat(filepath.Join(mount, "_tx")); !os.IsNotExist(err) {
		t.Fatalf("_tx should not be created on append failure, stat err=%v", err)
	}
}

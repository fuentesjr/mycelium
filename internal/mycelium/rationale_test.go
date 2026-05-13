package mycelium

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readLogRaw reads the first activity JSONL file and returns all non-empty lines as
// raw JSON maps, so tests can assert field presence without struct unmarshaling hiding omitted fields.
func readLogRaw(t *testing.T, mount string) []map[string]any {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl"))
	if err != nil {
		t.Fatalf("glob _activity: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no _activity/**/*.jsonl files found under %s", mount)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	var out []map[string]any
	for line := range strings.SplitSeq(string(data), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("unmarshal log line %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

// --- write --rationale ---

func TestWriteRationaleHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "content", "write", "notes.md", "--rationale", "explain write")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if rows[0]["rationale"] != "explain write" {
		t.Errorf("rationale: got %v, want %q", rows[0]["rationale"], "explain write")
	}
}

func TestWriteNoRationaleFieldAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "content", "write", "notes.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if _, ok := rows[0]["rationale"]; ok {
		t.Errorf("rationale key should be absent when not supplied, got %v", rows[0]["rationale"])
	}
}

func TestWriteRationaleOversizeRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	oversized := strings.Repeat("x", 64*1024+1)

	_, errOut, rc := runDispatchWithStdin(t, "content", "write", "notes.md", "--rationale", oversized)
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want ExitReservedPrefix (%d) (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if _, err := os.Stat(filepath.Join(mount, "notes.md")); !os.IsNotExist(err) {
		t.Error("file should not have been created on oversized rationale")
	}
	if logExists(mount) {
		t.Error("no activity entry should be written on oversized rationale")
	}
}

func TestWriteCASConflictWithRationale(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "v1")

	_, errOut, rc := runDispatchWithStdin(t, "v2", "write", "f.md",
		"--expected-version", "sha256:deadbeef",
		"--rationale", "why write")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want ExitConflict (stderr=%q)", rc, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Rationale != "why write" {
		t.Errorf("envelope rationale: got %q, want %q", env.Rationale, "why write")
	}
}

// --- edit --rationale ---

func TestEditRationaleHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "hello world")

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "world", "--new", "there", "--rationale", "explain edit")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if rows[0]["rationale"] != "explain edit" {
		t.Errorf("rationale: got %v, want %q", rows[0]["rationale"], "explain edit")
	}
}

func TestEditNoRationaleFieldAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "hello world")

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "world", "--new", "there")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if _, ok := rows[0]["rationale"]; ok {
		t.Errorf("rationale key should be absent when not supplied, got %v", rows[0]["rationale"])
	}
}

func TestEditRationaleOversizeRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "hello world")
	oversized := strings.Repeat("x", 64*1024+1)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "world", "--new", "there", "--rationale", oversized)
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want ExitReservedPrefix (%d) (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	// File must be unchanged.
	disk, _ := os.ReadFile(filepath.Join(mount, "f.md"))
	if string(disk) != "hello world" {
		t.Errorf("file should be unchanged on oversized rationale, got %q", disk)
	}
	if logExists(mount) {
		t.Error("no activity entry should be written on oversized rationale")
	}
}

func TestEditCASConflictWithRationale(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "hello world")

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md",
		"--old", "world", "--new", "there",
		"--expected-version", "sha256:deadbeef",
		"--rationale", "why edit")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want ExitConflict (stderr=%q)", rc, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Rationale != "why edit" {
		t.Errorf("envelope rationale: got %q, want %q", env.Rationale, "why edit")
	}
}

// --- rm --rationale ---

func TestRmRationaleHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "target.md", "to remove")

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "target.md", "--rationale", "explain rm")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if rows[0]["rationale"] != "explain rm" {
		t.Errorf("rationale: got %v, want %q", rows[0]["rationale"], "explain rm")
	}
}

func TestRmNoRationaleFieldAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "target.md", "to remove")

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "target.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if _, ok := rows[0]["rationale"]; ok {
		t.Errorf("rationale key should be absent when not supplied, got %v", rows[0]["rationale"])
	}
}

func TestRmRationaleOversizeRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "target.md", "content")
	oversized := strings.Repeat("x", 64*1024+1)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "target.md", "--rationale", oversized)
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want ExitReservedPrefix (%d) (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	// File must still exist.
	if _, err := os.Stat(filepath.Join(mount, "target.md")); err != nil {
		t.Errorf("file should still exist on oversized rationale: %v", err)
	}
	if logExists(mount) {
		t.Error("no activity entry should be written on oversized rationale")
	}
}

func TestRmCASConflictWithRationale(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "content")

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "f.md",
		"--expected-version", "sha256:deadbeef",
		"--rationale", "why rm")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want ExitConflict (stderr=%q)", rc, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Rationale != "why rm" {
		t.Errorf("envelope rationale: got %q, want %q", env.Rationale, "why rm")
	}
}

// --- mv --rationale ---

func TestMvRationaleHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "content")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md", "--rationale", "explain mv")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if rows[0]["rationale"] != "explain mv" {
		t.Errorf("rationale: got %v, want %q", rows[0]["rationale"], "explain mv")
	}
}

func TestMvNoRationaleFieldAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "content")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if _, ok := rows[0]["rationale"]; ok {
		t.Errorf("rationale key should be absent when not supplied, got %v", rows[0]["rationale"])
	}
}

func TestMvRationaleOversizeRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "content")
	oversized := strings.Repeat("x", 64*1024+1)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md", "--rationale", oversized)
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want ExitReservedPrefix (%d) (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	// src must be untouched.
	if _, err := os.Stat(filepath.Join(mount, "src.md")); err != nil {
		t.Errorf("src should still exist on oversized rationale: %v", err)
	}
	// dst must not be created.
	if _, err := os.Stat(filepath.Join(mount, "dst.md")); !os.IsNotExist(err) {
		t.Error("dst should not exist on oversized rationale")
	}
	if logExists(mount) {
		t.Error("no activity entry should be written on oversized rationale")
	}
}

func TestMvCASConflictWithRationale(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "content")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md",
		"--expected-version", "sha256:deadbeef",
		"--rationale", "why mv")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want ExitConflict (stderr=%q)", rc, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Rationale != "why mv" {
		t.Errorf("envelope rationale: got %q, want %q", env.Rationale, "why mv")
	}
}

func TestMvDestinationExistsWithRationale(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "source")
	writeTestFile(t, mount, "dst.md", "existing")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md", "--rationale", "why mv over existing")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want ExitConflict (stderr=%q)", rc, errOut)
	}
	env := parseDstExistsEnvelope(t, errOut)
	if env.Rationale != "why mv over existing" {
		t.Errorf("envelope rationale: got %q, want %q", env.Rationale, "why mv over existing")
	}
}

// --- log --rationale ---

func TestLogRationaleHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "log", "decision", "--rationale", "explain log")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if rows[0]["rationale"] != "explain log" {
		t.Errorf("rationale: got %v, want %q", rows[0]["rationale"], "explain log")
	}
}

func TestLogNoRationaleFieldAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "log", "decision")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := readLogRaw(t, mount)
	if len(rows) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(rows))
	}
	if _, ok := rows[0]["rationale"]; ok {
		t.Errorf("rationale key should be absent when not supplied, got %v", rows[0]["rationale"])
	}
}

func TestLogRationaleOversizeRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	oversized := strings.Repeat("x", 64*1024+1)

	_, errOut, rc := runDispatch(t, "log", "decision", "--rationale", oversized)
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want ExitReservedPrefix (%d) (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if logExists(mount) {
		t.Error("no activity entry should be written on oversized rationale")
	}
}

// --- Recovery with rationale in pending tx ---

func TestRecoveryPreservesRationaleInActivityEntry(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "agent")
	t.Setenv("MYCELIUM_SESSION_ID", "sess")
	id := Identity{AgentID: "agent", SessionID: "sess", Mount: mount}

	// Write a file whose committed version matches what we'll put in the pending tx.
	prior := versionPrefix + "absent"
	content := []byte("committed\n")
	post := hashVersion(content)
	writeTestFile(t, mount, "notes.md", string(content))

	// Build the pending tx manually with a rationale on the activity entry.
	txID := newULID()
	activity := LogEntry{
		Op:           "write",
		Path:         "notes.md",
		PriorVersion: prior,
		Version:      post,
		Rationale:    "recovery rationale",
	}
	tx := newContentTransaction(id, txID, fixedNow, activity, prior, post)
	writePendingTx(t, mount, tx)

	// Trigger recovery by issuing any mutation.
	_, errOut, rc := runDispatchWithStdin(t, "new\n", "write", "other.md")
	if rc != ExitOK {
		t.Fatalf("write after recovery rc=%d stderr=%q", rc, errOut)
	}

	// readLogLines reads all activity files across all dates.
	entries := readLogLines(t, mount)
	if len(entries) != 2 {
		t.Fatalf("log entries: got %d, want 2 (recovered + new write)", len(entries))
	}
	// The recovered entry is the one with the matching txID.
	var recovered *LogEntry
	for i := range entries {
		if entries[i].TxID == txID {
			recovered = &entries[i]
			break
		}
	}
	if recovered == nil {
		t.Fatalf("no recovered log entry found with tx_id %q", txID)
	}
	if recovered.Rationale != "recovery rationale" {
		t.Errorf("recovered entry rationale: got %q, want %q", recovered.Rationale, "recovery rationale")
	}
	if !recovered.Recovered {
		t.Error("recovered entry should have recovered=true")
	}
}

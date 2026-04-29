package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedNow is a stable timestamp used in tests that call appendLog directly.
var fixedNow = time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC)

// logJSONLPath returns the expected path of the log file under mount.
func logJSONLPath(mount string) string {
	return filepath.Join(mount, ".mycelium", "log.jsonl")
}

// readLogLines reads all lines from log.jsonl and returns them as parsed LogEntry values.
func readLogLines(t *testing.T, mount string) []LogEntry {
	t.Helper()
	data, err := os.ReadFile(logJSONLPath(mount))
	if err != nil {
		t.Fatalf("read log.jsonl: %v", err)
	}
	var entries []LogEntry
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var e LogEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("unmarshal log line %q: %v", line, err)
		}
		entries = append(entries, e)
	}
	return entries
}

func TestLogHappyPathNoPathNoPayload(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "agent-1", SessionID: "sess-1", Mount: mount}

	rc := appendLog(strings.NewReader(""), io.Discard, io.Discard, id, "context_signal", "", "", false, fixedNow)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "context_signal" {
		t.Errorf("op: got %q, want %q", e.Op, "context_signal")
	}
	if e.AgentID != "agent-1" {
		t.Errorf("agent_id: got %q, want %q", e.AgentID, "agent-1")
	}
	if e.SessionID != "sess-1" {
		t.Errorf("session_id: got %q, want %q", e.SessionID, "sess-1")
	}
	if e.TS == "" {
		t.Error("ts must not be empty")
	}
	if e.Path != "" {
		t.Errorf("path should be absent, got %q", e.Path)
	}
	if e.Payload != nil {
		t.Errorf("payload should be absent, got %s", e.Payload)
	}
}

func TestLogWithPath(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	rc := appendLog(strings.NewReader(""), io.Discard, io.Discard, id, "read", "memory.md", "", false, fixedNow)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Path != "memory.md" {
		t.Errorf("path: got %q, want %q", entries[0].Path, "memory.md")
	}
}

func TestLogWithPayloadJSON(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	rc := appendLog(strings.NewReader(""), io.Discard, io.Discard, id, "tool_call", "", `{"x":1}`, false, fixedNow)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	var got map[string]interface{}
	if err := json.Unmarshal(entries[0].Payload, &got); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if got["x"] != float64(1) {
		t.Errorf("payload x: got %v, want 1", got["x"])
	}
}

func TestLogWithStdin(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	rc := appendLog(strings.NewReader(`{"y":2}`), io.Discard, io.Discard, id, "tool_call", "", "", true, fixedNow)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	var got map[string]interface{}
	if err := json.Unmarshal(entries[0].Payload, &got); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if got["y"] != float64(2) {
		t.Errorf("payload y: got %v, want 2", got["y"])
	}
}

func TestLogBothPayloadJSONAndStdinIsUsageError(t *testing.T) {
	// The mutual-exclusion check lives in runLog (cli.go), so exercise it via dispatch.
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatchWithStdin(t, `{"z":3}`, "log", "op", "--payload-json", `{"a":1}`, "--stdin")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d", rc, ExitUsage)
	}
	if !strings.Contains(errOut, "mutually exclusive") {
		t.Errorf("stderr should mention mutual exclusion, got %q", errOut)
	}
	if _, err := os.Stat(logJSONLPath(mount)); !os.IsNotExist(err) {
		t.Error("log file should not be created on usage error")
	}
}

func TestLogInvalidPayloadJSON(t *testing.T) {
	mount := t.TempDir()
	var errBuf strings.Builder
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	rc := appendLog(strings.NewReader(""), io.Discard, &errBuf, id, "op", "", "not-json", false, fixedNow)
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d", rc, ExitUsage)
	}
	if _, err := os.Stat(logJSONLPath(mount)); !os.IsNotExist(err) {
		t.Error("log file should not be created on invalid JSON")
	}
}

func TestLogInvalidStdinJSON(t *testing.T) {
	mount := t.TempDir()
	var errBuf strings.Builder
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	rc := appendLog(strings.NewReader("not-json"), io.Discard, &errBuf, id, "op", "", "", true, fixedNow)
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d", rc, ExitUsage)
	}
	if _, err := os.Stat(logJSONLPath(mount)); !os.IsNotExist(err) {
		t.Error("log file should not be created on invalid stdin JSON")
	}
}

func TestLogMountUnset(t *testing.T) {
	var errBuf strings.Builder
	id := Identity{AgentID: "a", SessionID: "s", Mount: ""}

	rc := appendLog(strings.NewReader(""), io.Discard, &errBuf, id, "op", "", "", false, fixedNow)
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errBuf.String(), "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errBuf.String())
	}
}

func TestLogTwoAppendsTwoLines(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	if rc := appendLog(strings.NewReader(""), io.Discard, io.Discard, id, "first", "", "", false, fixedNow); rc != ExitOK {
		t.Fatalf("first append failed: rc=%d", rc)
	}
	if rc := appendLog(strings.NewReader(""), io.Discard, io.Discard, id, "second", "", "", false, fixedNow); rc != ExitOK {
		t.Fatalf("second append failed: rc=%d", rc)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Op != "first" {
		t.Errorf("first entry op: got %q, want %q", entries[0].Op, "first")
	}
	if entries[1].Op != "second" {
		t.Errorf("second entry op: got %q, want %q", entries[1].Op, "second")
	}

	// Verify the raw file has exactly 2 newline-terminated lines.
	raw, _ := os.ReadFile(logJSONLPath(mount))
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("raw file should have 2 lines, got %d", len(lines))
	}
}

func TestLogDotMyceliumDirAutoCreated(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	// Confirm .mycelium does not exist before the call.
	dotDir := filepath.Join(mount, ".mycelium")
	if _, err := os.Stat(dotDir); !os.IsNotExist(err) {
		t.Fatal(".mycelium should not exist before first log call")
	}

	if rc := appendLog(strings.NewReader(""), io.Discard, io.Discard, id, "op", "", "", false, fixedNow); rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	if _, err := os.Stat(dotDir); err != nil {
		t.Errorf(".mycelium dir should have been created: %v", err)
	}
}

func TestLogTimestampParsesAsRFC3339Nano(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	if rc := appendLog(strings.NewReader(""), io.Discard, io.Discard, id, "op", "", "", false, fixedNow); rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	parsed, err := time.Parse(time.RFC3339Nano, entries[0].TS)
	if err != nil {
		t.Fatalf("ts %q does not parse as RFC3339Nano: %v", entries[0].TS, err)
	}
	if !parsed.Equal(fixedNow.UTC()) {
		t.Errorf("ts round-trip: got %v, want %v", parsed, fixedNow.UTC())
	}
}

func TestLogE2EHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "e2e-agent")
	t.Setenv("MYCELIUM_SESSION_ID", "e2e-session")

	out, errOut, rc := runDispatch(t, "log", "memory_write", "--path", "notes.md", "--payload-json", `{"key":"val"}`)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	want := `{"log_status":"ok"}` + "\n"
	if out != want {
		t.Errorf("stdout: got %q, want %q", out, want)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "memory_write" {
		t.Errorf("op: got %q", e.Op)
	}
	if e.Path != "notes.md" {
		t.Errorf("path: got %q", e.Path)
	}
	if e.AgentID != "e2e-agent" {
		t.Errorf("agent_id: got %q", e.AgentID)
	}
	if e.SessionID != "e2e-session" {
		t.Errorf("session_id: got %q", e.SessionID)
	}
}

func TestLogE2EStdin(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatchWithStdin(t, `{"y":2}`, "log", "tool_call", "--stdin")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != `{"log_status":"ok"}`+"\n" {
		t.Errorf("stdout: got %q", out)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	var got map[string]interface{}
	if err := json.Unmarshal(entries[0].Payload, &got); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if got["y"] != float64(2) {
		t.Errorf("payload y: got %v, want 2", got["y"])
	}
}

func TestLogE2EMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")
	_, errOut, rc := runDispatch(t, "log", "op")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

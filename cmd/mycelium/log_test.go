package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// fixedNow is a stable timestamp used in tests that call appendLog/appendActivity directly.
var fixedNow = time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC)

// logExists returns true if any _activity/**/*.jsonl file exists under mount.
func logExists(mount string) bool {
	matches, _ := filepath.Glob(filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl"))
	return len(matches) > 0
}

// readLogLines finds all _activity/**/*.jsonl files under mount, sorts them,
// concatenates and parses all JSONL lines. Tests typically don't span days,
// so usually exactly one file is matched.
func readLogLines(t *testing.T, mount string) []LogEntry {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl"))
	if err != nil {
		t.Fatalf("glob _activity: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no _activity/**/*.jsonl files found under %s", mount)
	}
	sort.Strings(matches)

	var entries []LogEntry
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
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
	}
	return entries
}

// --- appendActivity / appendLog unit tests ---

func TestLogHappyPathNoPathNoPayload(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "agent-1", SessionID: "sess-1", Mount: mount}

	rc := appendActivity(io.Discard, io.Discard, id, LogEntry{Op: "context_signal"}, fixedNow)
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
	if len(e.Payload) != 0 {
		t.Errorf("payload should be absent, got %q", e.Payload)
	}
}

func TestLogWithPath(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	rc := appendActivity(io.Discard, io.Discard, id, LogEntry{Op: "read", Path: "memory.md"}, fixedNow)
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
		t.Fatalf("expected 1 activity entry, got %d", len(entries))
	}
	// Payload should be inlined on the entry.
	var got map[string]interface{}
	if err := json.Unmarshal(entries[0].Payload, &got); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if got["x"] != float64(1) {
		t.Errorf("payload x: got %v, want 1", got["x"])
	}

	// No logs/ directory should be created.
	logsDir := filepath.Join(mount, "logs")
	if _, err := os.Stat(logsDir); !os.IsNotExist(err) {
		t.Error("logs/ directory must not be created")
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

	// No logs/ directory should be created.
	logsDir := filepath.Join(mount, "logs")
	if _, err := os.Stat(logsDir); !os.IsNotExist(err) {
		t.Error("logs/ directory must not be created")
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
	if logExists(mount) {
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
	if logExists(mount) {
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
	if logExists(mount) {
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

	if rc := appendActivity(io.Discard, io.Discard, id, LogEntry{Op: "first"}, fixedNow); rc != ExitOK {
		t.Fatalf("first append failed: rc=%d", rc)
	}
	if rc := appendActivity(io.Discard, io.Discard, id, LogEntry{Op: "second"}, fixedNow); rc != ExitOK {
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
	logPath := activityLogPath(mount, "a", fixedNow)
	raw, _ := os.ReadFile(logPath)
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("raw file should have 2 lines, got %d", len(lines))
	}
}

func TestLogActivityDirAutoCreated(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	// Confirm _activity does not exist before the call.
	actDir := filepath.Join(mount, "_activity")
	if _, err := os.Stat(actDir); !os.IsNotExist(err) {
		t.Fatal("_activity should not exist before first log call")
	}

	if rc := appendActivity(io.Discard, io.Discard, id, LogEntry{Op: "op"}, fixedNow); rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	if _, err := os.Stat(actDir); err != nil {
		t.Errorf("_activity dir should have been created: %v", err)
	}
}

func TestLogTimestampParsesAsRFC3339Nano(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "a", SessionID: "s", Mount: mount}

	if rc := appendActivity(io.Discard, io.Discard, id, LogEntry{Op: "op"}, fixedNow); rc != ExitOK {
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
	// Payload should be inlined.
	var got map[string]interface{}
	if err := json.Unmarshal(e.Payload, &got); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if got["key"] != "val" {
		t.Errorf("payload key: got %v, want val", got["key"])
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

	// No logs/ directory should be created.
	logsDir := filepath.Join(mount, "logs")
	if _, err := os.Stat(logsDir); !os.IsNotExist(err) {
		t.Error("logs/ directory must not be created")
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

// --- New contract tests for inline-payload design ---

func TestLogNoPayloadNoPayloadField(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "log", "foo")
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Payload) != 0 {
		t.Errorf("payload should be absent when no payload given, got %q", entries[0].Payload)
	}

	// Verify the raw JSON also has no "payload" key.
	logPath := activityLogPath(mount, "", fixedNow)
	// Use glob to find the actual file since we don't control the timestamp.
	matches, _ := filepath.Glob(filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl"))
	if len(matches) == 0 {
		t.Fatal("no activity file found")
	}
	_ = logPath
	raw, _ := os.ReadFile(matches[0])
	if strings.Contains(string(raw), `"payload"`) {
		t.Errorf("raw JSONL should not contain payload key, got: %s", raw)
	}
}

func TestLogNeverCreatesLogsDir(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// With payload-json.
	runDispatch(t, "log", "sig", "--payload-json", `{"a":1}`)
	// With stdin.
	runDispatchWithStdin(t, `{"b":2}`, "log", "sig2", "--stdin")
	// Without payload.
	runDispatch(t, "log", "sig3")

	logsDir := filepath.Join(mount, "logs")
	if _, err := os.Stat(logsDir); !os.IsNotExist(err) {
		t.Errorf("logs/ must not be created by mycelium log, but it exists at %s", logsDir)
	}
}

func TestLogInvalidPayloadJSONViaDispatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, _, rc := runDispatch(t, "log", "op", "--payload-json", "not-json")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want ExitUsage", rc)
	}
	if logExists(mount) {
		t.Error("no entry should be written on invalid JSON")
	}
}

func TestLogInvalidStdinJSONViaDispatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, _, rc := runDispatchWithStdin(t, "not-json", "log", "op", "--stdin")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want ExitUsage", rc)
	}
	if logExists(mount) {
		t.Error("no entry should be written on invalid stdin JSON")
	}
}

// --- New activity log design tests ---

func TestActivityLogPathIsDateBucketed(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "test-agent")

	_, _, rc := runDispatchWithStdin(t, "hello", "write", "foo.md")
	if rc != ExitOK {
		t.Fatalf("write rc: got %d", rc)
	}

	matches, _ := filepath.Glob(filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl"))
	if len(matches) != 1 {
		t.Fatalf("expected exactly 1 _activity file, got %d: %v", len(matches), matches)
	}
	// Verify the path matches _activity/YYYY/MM/DD/test-agent.jsonl.
	rel, _ := filepath.Rel(filepath.Join(mount, "_activity"), matches[0])
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) != 4 {
		t.Fatalf("expected 4 path segments (YYYY/MM/DD/file), got %v", parts)
	}
	if parts[3] != "test-agent.jsonl" {
		t.Errorf("filename: got %q, want %q", parts[3], "test-agent.jsonl")
	}
}

func TestActivityLogUnspecifiedAgentIDFallback(t *testing.T) {
	mount := t.TempDir()
	id := Identity{AgentID: "", SessionID: "s", Mount: mount}

	rc := appendActivity(io.Discard, io.Discard, id, LogEntry{Op: "op"}, fixedNow)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}

	logPath := activityLogPath(mount, "", fixedNow)
	if !strings.HasSuffix(logPath, "unspecified.jsonl") {
		t.Errorf("expected filename unspecified.jsonl, got %q", logPath)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Errorf("expected file to exist at %q: %v", logPath, err)
	}
}

func TestActivityLogSchemaFlatForWrite(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	content := "hello\n"
	_, errOut, rc := runDispatchWithStdin(t, content, "write", "notes.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Version == "" {
		t.Error("version should be set for write entry")
	}
	if !strings.HasPrefix(e.Version, "sha256:") {
		t.Errorf("version should start with sha256:, got %q", e.Version)
	}
	wantVersion := sha256Hex(content)
	if e.Version != wantVersion {
		t.Errorf("version: got %q, want %q", e.Version, wantVersion)
	}
	if e.PriorVersion != "" {
		t.Errorf("prior_version should be absent for write, got %q", e.PriorVersion)
	}
	if e.From != "" {
		t.Errorf("from should be absent for write, got %q", e.From)
	}
	// Mutation entries must not have a payload field.
	if len(e.Payload) != 0 {
		t.Errorf("payload should be absent for mutation entries, got %q", e.Payload)
	}
}

func TestActivityLogSchemaFlatForRm(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "to remove\n"
	writeTestFile(t, mount, "target.md", content)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "target.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.PriorVersion == "" {
		t.Error("prior_version should be set for rm entry")
	}
	if !strings.HasPrefix(e.PriorVersion, "sha256:") {
		t.Errorf("prior_version should start with sha256:, got %q", e.PriorVersion)
	}
	wantVersion := sha256Hex(content)
	if e.PriorVersion != wantVersion {
		t.Errorf("prior_version: got %q, want %q", e.PriorVersion, wantVersion)
	}
	if e.Version != "" {
		t.Errorf("version should be absent for rm, got %q", e.Version)
	}
}

func TestActivityLogSchemaFlatForMv(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "move me\n"
	writeTestFile(t, mount, "src.md", content)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "mv" {
		t.Errorf("op: got %q, want mv", e.Op)
	}
	if e.Path != "dst.md" {
		t.Errorf("path should be dst, got %q", e.Path)
	}
	if e.From != "src.md" {
		t.Errorf("from should be src, got %q", e.From)
	}
	if e.Version == "" {
		t.Error("version should be set for mv entry")
	}
	wantVersion := sha256Hex(content)
	if e.Version != wantVersion {
		t.Errorf("version: got %q, want %q", e.Version, wantVersion)
	}
	if e.PriorVersion != "" {
		t.Errorf("prior_version should be absent for mv, got %q", e.PriorVersion)
	}
}

func TestLogPayloadInlinedOnEntry(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "pi-agent")

	payload := `{"x":42}`
	_, errOut, rc := runDispatch(t, "log", "context_signal", "--payload-json", payload)
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	var got map[string]interface{}
	if err := json.Unmarshal(entries[0].Payload, &got); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if got["x"] != float64(42) {
		t.Errorf("payload x: got %v, want 42", got["x"])
	}

	// Confirm no logs/ directory was created.
	if _, err := os.Stat(filepath.Join(mount, "logs")); !os.IsNotExist(err) {
		t.Error("logs/ directory must not exist")
	}
}

func TestLogNoPayloadAbsentOnEntry(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "log", "foo")
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if len(entries[0].Payload) != 0 {
		t.Errorf("payload should be absent when no payload given, got %q", entries[0].Payload)
	}
}

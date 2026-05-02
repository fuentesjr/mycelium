package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// evolveResult is parsed from stdout of a successful evolve call.
type evolveResult struct {
	ID         string `json:"id"`
	Supersedes string `json:"supersedes"`
}

// parseEvolveResult parses the JSON stdout of a successful evolve call.
func parseEvolveResult(t *testing.T, stdout string) evolveResult {
	t.Helper()
	line := strings.TrimRight(stdout, "\n")
	if idx := strings.Index(line, "\n"); idx >= 0 {
		line = line[:idx]
	}
	var r evolveResult
	if err := json.Unmarshal([]byte(line), &r); err != nil {
		t.Fatalf("parseEvolveResult: not valid JSON: %v\nstdout was: %q", err, stdout)
	}
	return r
}

// readAllEvolveEntries reads all evolve entries from the activity log.
func readAllEvolveEntries(t *testing.T, mount string) []evolveLogEntry {
	t.Helper()
	pattern := filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob activity: %v", err)
	}

	var entries []evolveLogEntry
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		sc.Buffer(make([]byte, 256*1024), 256*1024)
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				continue
			}
			var e evolveLogEntry
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				t.Fatalf("unmarshal line %q: %v", line, err)
			}
			if e.Op == "evolve" {
				entries = append(entries, e)
			}
		}
	}
	return entries
}

// ── Happy path ────────────────────────────────────────────────────────────────

func TestEvolveHappyPathBuiltinKind(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "agent-1")
	t.Setenv("MYCELIUM_SESSION_ID", "sess-1")

	out, errOut, rc := runDispatch(t, "evolve", "convention",
		"--target", "notes/incidents/",
		"--rationale", "Adopting date-slug naming.")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	res := parseEvolveResult(t, out)
	if res.ID == "" {
		t.Error("id should be non-empty")
	}
	if len(res.ID) != 26 {
		t.Errorf("id should be 26 chars (ULID), got %d: %q", len(res.ID), res.ID)
	}
	if res.Supersedes != "" {
		t.Errorf("supersedes should be absent on first write, got %q", res.Supersedes)
	}

	entries := readAllEvolveEntries(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 evolve entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "evolve" {
		t.Errorf("op: got %q, want evolve", e.Op)
	}
	if e.Kind != "convention" {
		t.Errorf("kind: got %q, want convention", e.Kind)
	}
	if e.Target != "notes/incidents/" {
		t.Errorf("target: got %q, want notes/incidents/", e.Target)
	}
	if e.Rationale != "Adopting date-slug naming." {
		t.Errorf("rationale: got %q", e.Rationale)
	}
	if e.ID != res.ID {
		t.Errorf("on-disk id %q != stdout id %q", e.ID, res.ID)
	}
	if e.Supersedes != "" {
		t.Errorf("on-disk supersedes should be absent, got %q", e.Supersedes)
	}
	if e.AgentID != "agent-1" {
		t.Errorf("agent_id: got %q", e.AgentID)
	}
	if e.SessionID != "sess-1" {
		t.Errorf("session_id: got %q", e.SessionID)
	}
	if e.TS == "" {
		t.Error("ts must not be empty")
	}
}

// ── Implicit supersession ─────────────────────────────────────────────────────

func TestEvolveImplicitSupersession(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// First write.
	out1, errOut1, rc1 := runDispatch(t, "evolve", "convention",
		"--target", "notes/incidents/",
		"--rationale", "First rule.")
	if rc1 != ExitOK {
		t.Fatalf("first evolve: rc=%d stderr=%q", rc1, errOut1)
	}
	res1 := parseEvolveResult(t, out1)

	// Second write with same (kind, target) — should supersede first.
	out2, errOut2, rc2 := runDispatch(t, "evolve", "convention",
		"--target", "notes/incidents/",
		"--rationale", "Updated rule.")
	if rc2 != ExitOK {
		t.Fatalf("second evolve: rc=%d stderr=%q", rc2, errOut2)
	}
	res2 := parseEvolveResult(t, out2)

	if res2.Supersedes != res1.ID {
		t.Errorf("second supersedes: got %q, want %q (first id)", res2.Supersedes, res1.ID)
	}
	if res2.ID == res1.ID {
		t.Error("second id must differ from first")
	}

	// On-disk verification.
	entries := readAllEvolveEntries(t, mount)
	if len(entries) != 2 {
		t.Fatalf("expected 2 evolve entries, got %d", len(entries))
	}
	if entries[1].Supersedes != res1.ID {
		t.Errorf("on-disk supersedes: got %q, want %q", entries[1].Supersedes, res1.ID)
	}
}

func TestEvolveCrossTargetNoSupersession(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	out1, _, rc1 := runDispatch(t, "evolve", "convention",
		"--target", "notes/incidents/",
		"--rationale", "Rule A.")
	if rc1 != ExitOK {
		t.Fatalf("first evolve: rc=%d", rc1)
	}
	res1 := parseEvolveResult(t, out1)

	// Different target — should not supersede.
	out2, _, rc2 := runDispatch(t, "evolve", "convention",
		"--target", "notes/decisions/",
		"--rationale", "Rule B.")
	if rc2 != ExitOK {
		t.Fatalf("second evolve: rc=%d", rc2)
	}
	res2 := parseEvolveResult(t, out2)

	if res2.Supersedes != "" {
		t.Errorf("different target must not auto-supersede, got supersedes=%q", res2.Supersedes)
	}
	_ = res1
}

// ── Explicit --supersedes ─────────────────────────────────────────────────────

func TestEvolveExplicitSupersedes(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Write first lesson (no target).
	out1, _, rc1 := runDispatch(t, "evolve", "lesson",
		"--rationale", "Old lesson.")
	if rc1 != ExitOK {
		t.Fatalf("first: rc=%d", rc1)
	}
	res1 := parseEvolveResult(t, out1)

	// Explicitly supersede it with a new lesson at a different target.
	out2, errOut2, rc2 := runDispatch(t, "evolve", "lesson",
		"--target", "some-topic",
		"--supersedes", res1.ID,
		"--rationale", "New lesson replacing the old one.")
	if rc2 != ExitOK {
		t.Fatalf("second: rc=%d stderr=%q", rc2, errOut2)
	}
	res2 := parseEvolveResult(t, out2)

	if res2.Supersedes != res1.ID {
		t.Errorf("supersedes: got %q, want %q", res2.Supersedes, res1.ID)
	}
}

func TestEvolveExplicitSupersedesMissingID(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolve", "lesson",
		"--supersedes", "01NONEXISTENTULID12345678",
		"--rationale", "Trying to supersede non-existent.")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "no such evolve event") {
		t.Errorf("stderr should mention 'no such evolve event', got %q", errOut)
	}
}

func TestEvolveExplicitSupersedesKindMismatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Write a lesson.
	out1, _, rc1 := runDispatch(t, "evolve", "lesson",
		"--rationale", "A lesson.")
	if rc1 != ExitOK {
		t.Fatalf("setup: rc=%d", rc1)
	}
	res1 := parseEvolveResult(t, out1)

	// Try to supersede it with a convention — kind mismatch.
	_, errOut, rc := runDispatch(t, "evolve", "convention",
		"--supersedes", res1.ID,
		"--rationale", "Wrong kind.")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "kind mismatch") {
		t.Errorf("stderr should mention 'kind mismatch', got %q", errOut)
	}
}

// ── First-use enforcement ─────────────────────────────────────────────────────

func TestEvolveFirstUseRequiresKindDefinition(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolve", "experiment",
		"--rationale", "Testing a new hypothesis.")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "first use") {
		t.Errorf("stderr should mention 'first use', got %q", errOut)
	}
	if !strings.Contains(errOut, "--kind-definition") {
		t.Errorf("stderr should mention '--kind-definition', got %q", errOut)
	}
}

func TestEvolveFirstUseWithKindDefinitionSucceeds(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "evolve", "experiment",
		"--kind-definition", "An in-progress hypothesis I am actively testing.",
		"--rationale", "Testing a new hypothesis.")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	res := parseEvolveResult(t, out)
	if res.ID == "" {
		t.Error("id should be non-empty")
	}

	// Also verify a _kind_definition event was written.
	entries := readAllEvolveEntries(t, mount)
	if len(entries) != 2 {
		t.Fatalf("expected 2 evolve entries (evolve + _kind_definition), got %d", len(entries))
	}
	// Find the _kind_definition entry.
	var kdEntry *evolveLogEntry
	for i := range entries {
		if entries[i].Kind == reservedKindDefinition {
			kdEntry = &entries[i]
			break
		}
	}
	if kdEntry == nil {
		t.Fatal("expected a _kind_definition entry in the log")
	}
	if kdEntry.Target != "experiment" {
		t.Errorf("_kind_definition target: got %q, want %q", kdEntry.Target, "experiment")
	}
	if kdEntry.Rationale != "An in-progress hypothesis I am actively testing." {
		t.Errorf("_kind_definition rationale: got %q", kdEntry.Rationale)
	}
}

func TestEvolveSecondUseOfAgentKindWithoutKindDefinitionSucceeds(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// First use — requires kind-definition.
	_, _, rc1 := runDispatch(t, "evolve", "experiment",
		"--kind-definition", "An in-progress hypothesis.",
		"--rationale", "First experiment.")
	if rc1 != ExitOK {
		t.Fatalf("first use: rc=%d", rc1)
	}

	// Second use — no kind-definition needed.
	out2, errOut2, rc2 := runDispatch(t, "evolve", "experiment",
		"--target", "hypotheses/x.md",
		"--rationale", "Second experiment.")
	if rc2 != ExitOK {
		t.Fatalf("second use: rc=%d stderr=%q", rc2, errOut2)
	}
	res2 := parseEvolveResult(t, out2)
	if res2.ID == "" {
		t.Error("id should be non-empty on second use")
	}
}

// ── Reserved _-prefix ─────────────────────────────────────────────────────────

func TestEvolveReservedUnderscorePrefixRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolve", "_internal",
		"--rationale", "Trying to use reserved prefix.")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention 'reserved', got %q", errOut)
	}
}

func TestEvolveReservedKindDefinitionKindRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// _kind_definition is the reserved meta-kind — agents cannot use it.
	_, errOut, rc := runDispatch(t, "evolve", "_kind_definition",
		"--rationale", "Should not work.")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
}

// ── Kind regex validation ─────────────────────────────────────────────────────

func TestEvolveKindRegexViolation(t *testing.T) {
	tests := []struct {
		kind string
		desc string
	}{
		{"kind!", "exclamation mark"},
		{"kind name", "space"},
		{"kind.name", "dot"},
		{"kind/name", "slash"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mount := t.TempDir()
			t.Setenv("MYCELIUM_MOUNT", mount)
			_, errOut, rc := runDispatch(t, "evolve", tt.kind, "--rationale", "test")
			if rc != ExitReservedPrefix {
				t.Errorf("kind %q: rc: got %d, want %d (stderr=%q)", tt.kind, rc, ExitReservedPrefix, errOut)
			}
		})
	}
}

func TestEvolveKindTooLong(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	longKind := strings.Repeat("a", 65) // 65 chars > max 64
	_, errOut, rc := runDispatch(t, "evolve", longKind, "--rationale", "test")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "exceeds") {
		t.Errorf("stderr should mention 'exceeds', got %q", errOut)
	}
}

func TestEvolveKindExactlyMaxLengthSucceeds(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	exactKind := strings.Repeat("a", 64)
	_, errOut, rc := runDispatch(t, "evolve", exactKind,
		"--kind-definition", "A kind with max-length name.",
		"--rationale", "test")
	if rc != ExitOK {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
}

// ── Rationale validation ──────────────────────────────────────────────────────

func TestEvolveEmptyRationaleRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolve", "lesson", "--rationale", "")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "rationale") {
		t.Errorf("stderr should mention 'rationale', got %q", errOut)
	}
}

func TestEvolveMissingRationaleRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolve", "lesson")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
}

func TestEvolveOversizeRationaleRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	bigRationale := strings.Repeat("x", 64*1024+1) // 64 KiB + 1 byte
	_, errOut, rc := runDispatch(t, "evolve", "lesson", "--rationale", bigRationale)
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "rationale") {
		t.Errorf("stderr should mention 'rationale', got %q", errOut)
	}
}

func TestEvolveExactlyMaxRationaleSucceeds(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	exactRationale := strings.Repeat("x", 64*1024)
	_, errOut, rc := runDispatch(t, "evolve", "lesson", "--rationale", exactRationale)
	if rc != ExitOK {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
}

// ── Target validation ─────────────────────────────────────────────────────────

func TestEvolveOversizeTargetRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	bigTarget := strings.Repeat("t", 4*1024+1) // 4 KiB + 1 byte
	_, errOut, rc := runDispatch(t, "evolve", "convention",
		"--target", bigTarget,
		"--rationale", "test")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "target") {
		t.Errorf("stderr should mention 'target', got %q", errOut)
	}
}

func TestEvolveExactlyMaxTargetSucceeds(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	exactTarget := strings.Repeat("t", 4*1024)
	_, errOut, rc := runDispatch(t, "evolve", "convention",
		"--target", exactTarget,
		"--rationale", "test")
	if rc != ExitOK {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
}

// ── Built-in redefinition ─────────────────────────────────────────────────────

func TestEvolveBuiltinRedefinitionSucceeds(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Redefining a built-in is allowed and should emit a _kind_definition event.
	out, errOut, rc := runDispatch(t, "evolve", "convention",
		"--kind-definition", "Redefined: a refined convention for this mount.",
		"--target", "notes/",
		"--rationale", "Broadening scope of convention to cover all notes.")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	res := parseEvolveResult(t, out)
	if res.ID == "" {
		t.Error("id should be non-empty")
	}

	entries := readAllEvolveEntries(t, mount)
	// Should have: 1 evolve + 1 _kind_definition = 2.
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (evolve + _kind_definition), got %d", len(entries))
	}
	var kdEntry *evolveLogEntry
	for i := range entries {
		if entries[i].Kind == reservedKindDefinition {
			kdEntry = &entries[i]
		}
	}
	if kdEntry == nil {
		t.Fatal("missing _kind_definition entry")
	}
	if kdEntry.Target != "convention" {
		t.Errorf("_kind_definition target: got %q, want convention", kdEntry.Target)
	}
}

func TestEvolveAgentKindRedefinitionChain(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// First use of custom kind with definition.
	_, _, rc1 := runDispatch(t, "evolve", "hypothesis",
		"--kind-definition", "First definition.",
		"--rationale", "Introduce hypothesis kind.")
	if rc1 != ExitOK {
		t.Fatalf("first: rc=%d", rc1)
	}

	// Redefine it.
	_, errOut2, rc2 := runDispatch(t, "evolve", "hypothesis",
		"--target", "hyp/x.md",
		"--kind-definition", "Second definition, refined.",
		"--rationale", "Redefining hypothesis kind.")
	if rc2 != ExitOK {
		t.Fatalf("redefine: rc=%d stderr=%q", rc2, errOut2)
	}

	entries := readAllEvolveEntries(t, mount)
	// Expected: evolve1 + _kd1 + evolve2 + _kd2 = 4 entries.
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	// Find the two _kind_definition entries.
	var kdEntries []evolveLogEntry
	for _, e := range entries {
		if e.Kind == reservedKindDefinition {
			kdEntries = append(kdEntries, e)
		}
	}
	if len(kdEntries) != 2 {
		t.Fatalf("expected 2 _kind_definition entries, got %d", len(kdEntries))
	}

	// The second _kind_definition should supersede the first.
	kd1 := kdEntries[0]
	kd2 := kdEntries[1]
	if kd2.Supersedes != kd1.ID {
		t.Errorf("second _kind_definition supersedes: got %q, want %q (first _kd id)", kd2.Supersedes, kd1.ID)
	}
	if kd2.Supersedes == "" {
		t.Error("second _kind_definition must supersede the first")
	}
}

// ── On-disk JSONL schema ──────────────────────────────────────────────────────

func TestEvolveOnDiskSchema(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "researcher-7")
	t.Setenv("MYCELIUM_SESSION_ID", "abc123")

	out, errOut, rc := runDispatch(t, "evolve", "convention",
		"--target", "notes/incidents/",
		"--rationale", "Adopting date-slug filenames.")
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}
	res := parseEvolveResult(t, out)

	// Read the raw JSONL file and verify field shape.
	pattern := filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl")
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		t.Fatal("no activity file found")
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read activity file: %v", err)
	}

	// Parse the raw line.
	line := strings.TrimRight(string(raw), "\n")
	var flat map[string]interface{}
	if err := json.Unmarshal([]byte(line), &flat); err != nil {
		t.Fatalf("parse raw line: %v", err)
	}

	// Required fields.
	assertStringField := func(field string) string {
		t.Helper()
		v, ok := flat[field]
		if !ok {
			t.Errorf("field %q missing from on-disk entry", field)
			return ""
		}
		s, ok := v.(string)
		if !ok {
			t.Errorf("field %q: expected string, got %T", field, v)
			return ""
		}
		return s
	}

	if ts := assertStringField("ts"); ts == "" {
		t.Error("ts must be non-empty")
	}
	if op := assertStringField("op"); op != "evolve" {
		t.Errorf("op: got %q, want evolve", op)
	}
	if id := assertStringField("id"); id != res.ID {
		t.Errorf("id: got %q, want %q", id, res.ID)
	}
	if kind := assertStringField("kind"); kind != "convention" {
		t.Errorf("kind: got %q, want convention", kind)
	}
	if target := assertStringField("target"); target != "notes/incidents/" {
		t.Errorf("target: got %q", target)
	}
	if rationale := assertStringField("rationale"); rationale != "Adopting date-slug filenames." {
		t.Errorf("rationale: got %q", rationale)
	}
	if agentID := assertStringField("agent_id"); agentID != "researcher-7" {
		t.Errorf("agent_id: got %q", agentID)
	}
	if sessionID := assertStringField("session_id"); sessionID != "abc123" {
		t.Errorf("session_id: got %q", sessionID)
	}

	// Optional fields absent when empty.
	if _, ok := flat["supersedes"]; ok {
		t.Error("supersedes field must be absent when empty")
	}
	if _, ok := flat["kind_definition"]; ok {
		t.Error("kind_definition field must be absent when not supplied")
	}
	if _, ok := flat["payload"]; ok {
		t.Error("payload field must be absent for evolve entries")
	}
}

// ── Mount env check ───────────────────────────────────────────────────────────

func TestEvolveMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")
	_, errOut, rc := runDispatch(t, "evolve", "lesson", "--rationale", "test")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

// ── Kind missing positional ───────────────────────────────────────────────────

func TestEvolveMissingKindPositional(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolve", "--rationale", "test")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
}

// ── No target: no implicit supersession ──────────────────────────────────────

func TestEvolveNoTargetLessonNoSupersession(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Two lessons with no target — same (kind="lesson", target="") pair.
	// Second should implicitly supersede first.
	out1, _, rc1 := runDispatch(t, "evolve", "lesson", "--rationale", "First lesson.")
	if rc1 != ExitOK {
		t.Fatalf("first: rc=%d", rc1)
	}
	res1 := parseEvolveResult(t, out1)

	out2, _, rc2 := runDispatch(t, "evolve", "lesson", "--rationale", "Second lesson.")
	if rc2 != ExitOK {
		t.Fatalf("second: rc=%d", rc2)
	}
	res2 := parseEvolveResult(t, out2)

	// Both have target="", so second supersedes first.
	if res2.Supersedes != res1.ID {
		t.Errorf("supersedes: got %q, want %q", res2.Supersedes, res1.ID)
	}
}

// ── Concurrency smoke test ────────────────────────────────────────────────────

// TestEvolveConcurrentSameKindTarget spawns N child processes all writing
// evolve events for the same (kind, target). Under the mount lock, the
// supersession chain must be monotonic: exactly one event has no supersedes,
// every other event supersedes a predecessor, and there are no cycles.
func TestEvolveConcurrentSameKindTarget(t *testing.T) {
	mount := t.TempDir()

	const N = 6
	type result struct {
		stdout string
		stderr string
		code   int
		err    error
	}
	ch := make(chan result, N)

	for i := 0; i < N; i++ {
		go func(idx int) {
			stdout, stderr, code, err := runMyceliumChild(mount, "",
				"evolve", "convention",
				"--target", "shared-target",
				"--rationale", fmt.Sprintf("Concurrent writer %d", idx),
			)
			ch <- result{stdout, stderr, code, err}
		}(i)
	}

	var successes int
	for i := 0; i < N; i++ {
		r := <-ch
		if r.err != nil {
			t.Errorf("child run error: %v", r.err)
			continue
		}
		if r.code != ExitOK {
			t.Errorf("child failed: code=%d stderr=%q", r.code, r.stderr)
			continue
		}
		successes++
	}

	if successes != N {
		t.Fatalf("expected all %d evolve calls to succeed, got %d successes", N, successes)
	}

	// Load all entries and verify the supersession chain is valid (no cycles, monotonic).
	entries := readAllEvolveEntries(t, mount)
	// Filter to only the convention+shared-target entries.
	var chain []evolveLogEntry
	for _, e := range entries {
		if e.Kind == "convention" && e.Target == "shared-target" {
			chain = append(chain, e)
		}
	}
	if len(chain) != N {
		t.Fatalf("expected %d chain entries, got %d", N, len(chain))
	}

	// Build ID set and supersedes map.
	idSet := make(map[string]bool)
	for _, e := range chain {
		idSet[e.ID] = true
	}
	supersedesMap := make(map[string]string) // supersedes -> id
	for _, e := range chain {
		if e.Supersedes != "" {
			supersedesMap[e.ID] = e.Supersedes
		}
	}

	// Exactly one entry must have no supersedes (the first in the chain).
	var roots []string
	for _, e := range chain {
		if e.Supersedes == "" {
			roots = append(roots, e.ID)
		}
	}
	if len(roots) != 1 {
		t.Errorf("expected exactly 1 root (no supersedes), got %d: %v", len(roots), roots)
	}

	// All supersedes targets must reference valid IDs within the set.
	for id, sup := range supersedesMap {
		if !idSet[sup] {
			t.Errorf("entry %q supersedes %q which is not in the chain", id, sup)
		}
	}

	// No cycles: walk from each entry following supersedes; must terminate.
	for _, e := range chain {
		seen := make(map[string]bool)
		cur := e.ID
		for cur != "" {
			if seen[cur] {
				t.Errorf("cycle detected at %q", cur)
				break
			}
			seen[cur] = true
			cur = supersedesMap[cur]
		}
	}
}

// ── Hyphen in kind name is valid ──────────────────────────────────────────────

func TestEvolveHyphenInKindIsValid(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolve", "dead-end",
		"--kind-definition", "A path that didn't work out.",
		"--rationale", "Tried approach X, it failed.")
	if rc != ExitOK {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
}

// ── evolve appears in subcommand list ─────────────────────────────────────────

func TestEvolveAppearsInSubcommandList(t *testing.T) {
	_, errOut, _ := runDispatch(t, "nope")
	if !strings.Contains(errOut, "evolve") {
		t.Errorf("'evolve' not found in subcommand list: %q", errOut)
	}
}

package main

import (
	"bufio"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// parseEvolutionKindRows parses the JSON array output of `evolution --kinds`.
func parseEvolutionKindRows(t *testing.T, stdout string) []evolutionKindRow {
	t.Helper()
	line := strings.TrimRight(stdout, "\n")
	var rows []evolutionKindRow
	if err := json.Unmarshal([]byte(line), &rows); err != nil {
		t.Fatalf("parseEvolutionKindRows: not valid JSON array: %v\nstdout was: %q", err, stdout)
	}
	return rows
}

// parseEvolutionEntries parses newline-delimited JSON entries from evolution default / --active output.
func parseEvolutionEntries(t *testing.T, stdout string) []evolveLogEntry {
	t.Helper()
	var entries []evolveLogEntry
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var e evolveLogEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("parseEvolutionEntries: invalid JSON line %q: %v", line, err)
		}
		entries = append(entries, e)
	}
	return entries
}

// seedEvolve is a convenience wrapper that calls runEvolve via dispatch and
// fatals on failure. It returns the parsed result (id, supersedes).
func seedEvolve(t *testing.T, args ...string) evolveResult {
	t.Helper()
	out, errOut, rc := runDispatch(t, append([]string{"evolve"}, args...)...)
	if rc != ExitOK {
		t.Fatalf("seedEvolve: rc=%d stderr=%q args=%v", rc, errOut, args)
	}
	return parseEvolveResult(t, out)
}

// ── Test 1: --kinds on empty mount ────────────────────────────────────────────

func TestEvolutionKindsEmptyMount(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "evolution", "--kinds")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := parseEvolutionKindRows(t, out)
	if len(rows) != 5 {
		t.Fatalf("expected 5 builtin rows, got %d: %v", len(rows), rows)
	}

	wantNames := []string{"convention", "index", "archive", "lesson", "question"}
	for i, want := range wantNames {
		r := rows[i]
		if r.Name != want {
			t.Errorf("row[%d].name: got %q, want %q", i, r.Name, want)
		}
		if r.Source != "builtin" {
			t.Errorf("row[%d].source: got %q, want builtin", i, r.Source)
		}
		if r.EventCount != 0 {
			t.Errorf("row[%d].event_count: got %d, want 0", i, r.EventCount)
		}
		if r.Definition == "" {
			t.Errorf("row[%d].definition: must be non-empty", i)
		}
		if r.DefinedAtVersion == "" {
			t.Errorf("row[%d].defined_at_version: must be non-empty", i)
		}
	}
}

// ── Test 2: default mode on empty mount ──────────────────────────────────────

func TestEvolutionDefaultEmptyMount(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "evolution")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("stdout: got %q, want empty", out)
	}
}

// ── Test 3: --active on empty mount ──────────────────────────────────────────

func TestEvolutionActiveEmptyMount(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "evolution", "--active")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("stdout: got %q, want empty", out)
	}
}

// ── Test 4: default mode returns multiple events in chronological order ───────

func TestEvolutionDefaultMultipleEvents(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "agent-1")
	t.Setenv("MYCELIUM_SESSION_ID", "sess-1")

	r1 := seedEvolve(t, "convention", "--target", "notes/", "--rationale", "First convention.")
	time.Sleep(time.Millisecond)
	r2 := seedEvolve(t, "lesson", "--rationale", "A lesson learned.")
	time.Sleep(time.Millisecond)
	r3 := seedEvolve(t, "index", "--target", "idx/", "--rationale", "Built an index.")

	out, errOut, rc := runDispatch(t, "evolution")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := parseEvolutionEntries(t, out)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Chronological (ULID) order.
	if entries[0].ID != r1.ID {
		t.Errorf("entries[0].id: got %q, want %q", entries[0].ID, r1.ID)
	}
	if entries[1].ID != r2.ID {
		t.Errorf("entries[1].id: got %q, want %q", entries[1].ID, r2.ID)
	}
	if entries[2].ID != r3.ID {
		t.Errorf("entries[2].id: got %q, want %q", entries[2].ID, r3.ID)
	}

	// Each line is independently parseable (already checked above), so verify
	// key fields on one entry for completeness.
	if entries[0].Kind != "convention" {
		t.Errorf("entries[0].kind: got %q, want convention", entries[0].Kind)
	}
	if entries[0].Rationale != "First convention." {
		t.Errorf("entries[0].rationale: got %q", entries[0].Rationale)
	}
}

// ── Test 5: --kind filter ─────────────────────────────────────────────────────

func TestEvolutionKindFilter(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	seedEvolve(t, "convention", "--target", "notes/", "--rationale", "A convention.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "lesson", "--rationale", "A lesson.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "convention", "--target", "docs/", "--rationale", "Another convention.")

	out, errOut, rc := runDispatch(t, "evolution", "--kind", "convention")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := parseEvolutionEntries(t, out)
	if len(entries) != 2 {
		t.Fatalf("expected 2 convention entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Kind != "convention" {
			t.Errorf("expected kind=convention, got %q", e.Kind)
		}
	}
}

// ── Test 6: --since RFC3339 filter ───────────────────────────────────────────

func TestEvolutionSinceRFC3339(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Seed both events with a guaranteed time gap.
	seedEvolve(t, "lesson", "--target", "early", "--rationale", "Early lesson.")
	time.Sleep(2 * time.Millisecond)
	r2 := seedEvolve(t, "lesson", "--target", "late", "--rationale", "Late lesson.")

	// Read both events from disk to get their actual on-disk timestamps.
	allEntries := readAllEvolveEntries(t, mount)
	if len(allEntries) != 2 {
		t.Fatalf("setup: expected 2 on-disk entries, got %d", len(allEntries))
	}

	// Parse the second event's ts; use it as the --since boundary.
	// This guarantees we get exactly 1 result: the second event is included
	// (ts >= ts), and the first is excluded (ts < ts of second).
	var ts2 time.Time
	for _, e := range allEntries {
		if e.ID == r2.ID {
			var err error
			ts2, err = time.Parse(time.RFC3339Nano, e.TS)
			if err != nil {
				ts2, err = time.Parse(time.RFC3339, e.TS)
				if err != nil {
					t.Fatalf("parse ts of second event: %v", err)
				}
			}
			break
		}
	}
	if ts2.IsZero() {
		t.Fatal("could not find second event's timestamp on disk")
	}

	sinceStr := ts2.UTC().Format(time.RFC3339Nano)
	out, errOut, rc := runDispatch(t, "evolution", "--since", sinceStr)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := parseEvolutionEntries(t, out)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after since filter, got %d (entries: %v)", len(entries), entries)
	}
	if entries[0].ID != r2.ID {
		t.Errorf("entry id: got %q, want %q", entries[0].ID, r2.ID)
	}
}

// ── Test 7: --since YYYY-MM-DD accepted as midnight UTC ──────────────────────

func TestEvolutionSinceDateOnly(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	seedEvolve(t, "lesson", "--rationale", "Some lesson.")

	// Use a far-future date — should return no events.
	out, errOut, rc := runDispatch(t, "evolution", "--since", "2099-01-01")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("expected no output for far-future --since, got %q", out)
	}

	// Use a date in the past — should return the event.
	out2, errOut2, rc2 := runDispatch(t, "evolution", "--since", "2000-01-01")
	if rc2 != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc2, ExitOK, errOut2)
	}
	entries := parseEvolutionEntries(t, out2)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for past --since, got %d", len(entries))
	}
}

// ── Test 8: --since invalid format → exit 2 ──────────────────────────────────

func TestEvolutionSinceInvalidFormat(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolution", "--since", "not-a-date")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if errOut == "" {
		t.Error("expected error on stderr for invalid --since format")
	}
}

// ── Test 9: --active collapses chains ─────────────────────────────────────────

func TestEvolutionActiveCollapsesChains(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	seedEvolve(t, "convention", "--target", "foo", "--rationale", "First foo convention.")
	time.Sleep(time.Millisecond)
	r2 := seedEvolve(t, "convention", "--target", "foo", "--rationale", "Updated foo convention.")

	out, errOut, rc := runDispatch(t, "evolution", "--active")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := parseEvolutionEntries(t, out)
	if len(entries) != 1 {
		t.Fatalf("expected 1 active entry (latest), got %d", len(entries))
	}
	if entries[0].ID != r2.ID {
		t.Errorf("active entry id: got %q, want %q (second/latest)", entries[0].ID, r2.ID)
	}
}

// ── Test 10: --active with multiple distinct (kind, target) ──────────────────

func TestEvolutionActiveMultipleDistinctPairs(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	r1 := seedEvolve(t, "convention", "--target", "foo", "--rationale", "Convention foo.")
	time.Sleep(time.Millisecond)
	r2 := seedEvolve(t, "convention", "--target", "bar", "--rationale", "Convention bar.")
	time.Sleep(time.Millisecond)
	r3 := seedEvolve(t, "lesson", "--target", "baz", "--rationale", "Lesson baz.")

	out, errOut, rc := runDispatch(t, "evolution", "--active")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := parseEvolutionEntries(t, out)
	if len(entries) != 3 {
		t.Fatalf("expected 3 active entries, got %d", len(entries))
	}

	// Build a set of returned IDs for order-independent assertion.
	idSet := make(map[string]bool)
	for _, e := range entries {
		idSet[e.ID] = true
	}
	for _, wantID := range []string{r1.ID, r2.ID, r3.ID} {
		if !idSet[wantID] {
			t.Errorf("expected active entry with id %q, not found in output", wantID)
		}
	}
}

// ── Test 11: --kinds distinguishes builtin/agent ──────────────────────────────

func TestEvolutionKindsDistinguishesBuiltinAgent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	const expDef = "An in-progress hypothesis I am actively testing."
	seedEvolve(t, "convention", "--target", "notes/", "--rationale", "A convention.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "experiment",
		"--kind-definition", expDef,
		"--rationale", "First experiment.")

	out, errOut, rc := runDispatch(t, "evolution", "--kinds")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := parseEvolutionKindRows(t, out)

	// Find convention row.
	var conventionRow, experimentRow *evolutionKindRow
	for i := range rows {
		switch rows[i].Name {
		case "convention":
			conventionRow = &rows[i]
		case "experiment":
			experimentRow = &rows[i]
		}
	}

	if conventionRow == nil {
		t.Fatal("convention row not found in --kinds output")
	}
	if conventionRow.Source != "builtin" {
		t.Errorf("convention source: got %q, want builtin", conventionRow.Source)
	}
	// Built-in definition should be the canonical one from kinds.go.
	if conventionRow.Definition == "" {
		t.Error("convention definition must be non-empty")
	}
	if conventionRow.DefinedAtVersion == "" {
		t.Error("convention defined_at_version must be non-empty")
	}

	if experimentRow == nil {
		t.Fatal("experiment row not found in --kinds output")
	}
	if experimentRow.Source != "agent" {
		t.Errorf("experiment source: got %q, want agent", experimentRow.Source)
	}
	if experimentRow.Definition != expDef {
		t.Errorf("experiment definition: got %q, want %q", experimentRow.Definition, expDef)
	}
}

// ── Test 12: --kinds event_count ─────────────────────────────────────────────

func TestEvolutionKindsEventCount(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// 3 convention events (different targets so no supersession collapse).
	seedEvolve(t, "convention", "--target", "a/", "--rationale", "Convention A.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "convention", "--target", "b/", "--rationale", "Convention B.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "convention", "--target", "c/", "--rationale", "Convention C.")
	time.Sleep(time.Millisecond)
	// 1 lesson event.
	seedEvolve(t, "lesson", "--rationale", "A lesson.")

	out, errOut, rc := runDispatch(t, "evolution", "--kinds")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := parseEvolutionKindRows(t, out)

	var conventionCount, lessonCount int
	for _, r := range rows {
		switch r.Name {
		case "convention":
			conventionCount = r.EventCount
		case "lesson":
			lessonCount = r.EventCount
		}
	}

	if conventionCount != 3 {
		t.Errorf("convention event_count: got %d, want 3", conventionCount)
	}
	if lessonCount != 1 {
		t.Errorf("lesson event_count: got %d, want 1", lessonCount)
	}
}

// ── Test 13: --kinds excludes _kind_definition from event_count ───────────────

func TestEvolutionKindsExcludesKindDefinitionFromCount(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Introduce agent kind "experiment" (writes 1 evolve + 1 _kind_definition).
	seedEvolve(t, "experiment",
		"--kind-definition", "Hypothesis under test.",
		"--rationale", "First experiment.")
	time.Sleep(time.Millisecond)
	// Second event of same kind at different target.
	seedEvolve(t, "experiment",
		"--target", "hyp/x.md",
		"--rationale", "Second experiment.")
	time.Sleep(time.Millisecond)
	// Redefine the kind (writes another evolve + another _kind_definition).
	seedEvolve(t, "experiment",
		"--target", "hyp/y.md",
		"--kind-definition", "Redefined hypothesis.",
		"--rationale", "Third experiment.")

	out, errOut, rc := runDispatch(t, "evolution", "--kinds")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := parseEvolutionKindRows(t, out)
	var experimentRow *evolutionKindRow
	for i := range rows {
		if rows[i].Name == "experiment" {
			experimentRow = &rows[i]
		}
	}
	if experimentRow == nil {
		t.Fatal("experiment row not found in --kinds output")
	}
	// 3 real evolve events; the 2 synthetic _kind_definition events must NOT be counted.
	if experimentRow.EventCount != 3 {
		t.Errorf("experiment event_count: got %d, want 3 (synthetics must not count)", experimentRow.EventCount)
	}
}

// ── Test 14: --kinds returns updated definition after redefinition ─────────────

func TestEvolutionKindsShowsLatestDefinition(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	const defA = "Original definition of experiment."
	const defB = "Refined definition of experiment."

	seedEvolve(t, "experiment",
		"--kind-definition", defA,
		"--rationale", "First use.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "experiment",
		"--target", "hyp/x.md",
		"--kind-definition", defB,
		"--rationale", "Redefine.")

	out, errOut, rc := runDispatch(t, "evolution", "--kinds")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	rows := parseEvolutionKindRows(t, out)
	var experimentRow *evolutionKindRow
	for i := range rows {
		if rows[i].Name == "experiment" {
			experimentRow = &rows[i]
		}
	}
	if experimentRow == nil {
		t.Fatal("experiment row not found in --kinds output")
	}
	if experimentRow.Definition != defB {
		t.Errorf("experiment definition: got %q, want %q (latest definition)", experimentRow.Definition, defB)
	}
}

// ── Test 15: default mode excludes _kind_definition events ────────────────────

func TestEvolutionDefaultExcludesKindDefinition(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Introduce and redefine an agent kind to generate _kind_definition synthetics.
	seedEvolve(t, "experiment",
		"--kind-definition", "Initial.",
		"--rationale", "First.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "experiment",
		"--target", "hyp/x.md",
		"--kind-definition", "Redefined.",
		"--rationale", "Second.")

	out, errOut, rc := runDispatch(t, "evolution")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := parseEvolutionEntries(t, out)
	for _, e := range entries {
		if e.Kind == reservedKindDefinition {
			t.Errorf("default mode emitted a %q entry: %+v", reservedKindDefinition, e)
		}
	}
	// Should have exactly the 2 real evolve events.
	if len(entries) != 2 {
		t.Errorf("expected 2 user-facing entries, got %d", len(entries))
	}
}

// ── Test 16: mutually exclusive flags ────────────────────────────────────────

func TestEvolutionMutuallyExclusiveFlags(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	tests := []struct {
		name string
		args []string
	}{
		{"active+kinds", []string{"evolution", "--active", "--kinds"}},
		{"kinds+kind", []string{"evolution", "--kinds", "--kind", "convention"}},
		{"kinds+since", []string{"evolution", "--kinds", "--since", "2026-01-01"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errOut, rc := runDispatch(t, tt.args...)
			if rc != ExitUsage {
				t.Errorf("%s: rc: got %d, want %d (stderr=%q)", tt.name, rc, ExitUsage, errOut)
			}
			if errOut == "" {
				t.Errorf("%s: expected error on stderr", tt.name)
			}
		})
	}
}

// ── Test 17: invalid --format ─────────────────────────────────────────────────

func TestEvolutionInvalidFormat(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "evolution", "--format", "xml")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if errOut == "" {
		t.Error("expected error on stderr for invalid --format")
	}
}

// ── Test 18: --format text smoke ─────────────────────────────────────────────

func TestEvolutionFormatTextSmoke(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	const rationale = "Adopting date-slug naming."
	seedEvolve(t, "convention",
		"--target", "notes/incidents/",
		"--rationale", rationale)

	out, errOut, rc := runDispatch(t, "evolution", "--format", "text")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	// Output should NOT be JSON.
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Error("--format text output looks like JSON; expected non-JSON format")
	}
	// Should contain the kind name and at least part of the rationale.
	if !strings.Contains(out, "convention") {
		t.Errorf("text output missing 'convention': %q", out)
	}
	// The rationale first line must appear somewhere.
	if !strings.Contains(out, "Adopting date-slug naming.") {
		t.Errorf("text output missing rationale fragment: %q", out)
	}
}

// ── Test 19: JSON output is newline-delimited (one object per line) ───────────

func TestEvolutionJSONNewlineDelimited(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	seedEvolve(t, "convention", "--target", "a/", "--rationale", "Convention A.")
	time.Sleep(time.Millisecond)
	seedEvolve(t, "lesson", "--rationale", "Lesson.")

	// Default mode.
	outDefault, errOut, rc := runDispatch(t, "evolution")
	if rc != ExitOK {
		t.Fatalf("default: rc=%d stderr=%q", rc, errOut)
	}
	lines := strings.Split(strings.TrimRight(outDefault, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("default: expected 2 lines, got %d: %q", len(lines), outDefault)
	}
	for i, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("default line %d is not valid JSON: %v: %q", i, err, line)
		}
	}

	// --active mode.
	outActive, errOut2, rc2 := runDispatch(t, "evolution", "--active")
	if rc2 != ExitOK {
		t.Fatalf("active: rc=%d stderr=%q", rc2, errOut2)
	}
	activeLines := strings.Split(strings.TrimRight(outActive, "\n"), "\n")
	if len(activeLines) != 2 {
		t.Fatalf("active: expected 2 lines (distinct targets), got %d: %q", len(activeLines), outActive)
	}
	for i, line := range activeLines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("active line %d is not valid JSON: %v: %q", i, err, line)
		}
	}
}

// ── Test 20: MYCELIUM_MOUNT unset ────────────────────────────────────────────

func TestEvolutionMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")

	_, errOut, rc := runDispatch(t, "evolution")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

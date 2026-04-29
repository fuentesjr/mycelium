package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ---- Direct helper tests ----

func TestGrepDirectHelper(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "hello.md", "hello world\n")

	var out, errOut bytes.Buffer
	opts := GrepOptions{
		Pattern: "world",
		Format:  "text",
		Limit:   1000,
	}
	rc := grepFiles(&out, &errOut, mount, opts)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "hello.md:1:hello world") {
		t.Errorf("output missing expected match, got %q", got)
	}
}

// ---- Happy path tests ----

func TestGrepLiteralMatch(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "notes.md", "line one\nfoo bar baz\nline three\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "foo bar")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	want := "notes.md:2:foo bar baz\n"
	if out != want {
		t.Errorf("stdout: got %q, want %q", out, want)
	}
}

func TestGrepRegexMatch(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "words.txt", "foo\nfoo bar\nfuuo\nf.+o\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Pattern f.+o matches "foo", "foo bar", "fuuo", "f.+o"
	out, errOut, rc := runDispatch(t, "grep", "--pattern", "f.+o", "--regex")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 matches, got %d: %q", len(lines), out)
	}
	if !strings.Contains(out, "words.txt:1:foo") {
		t.Errorf("missing match for line 1, got %q", out)
	}
	if !strings.Contains(out, "words.txt:2:foo bar") {
		t.Errorf("missing match for line 2, got %q", out)
	}
}

func TestGrepMultipleMatchesAcrossFiles(t *testing.T) {
	mount := t.TempDir()
	// Files created in non-alphabetical order; walk (and output) must be sorted by path.
	mkfile(t, mount, "zebra.md", "needle\n")
	mkfile(t, mount, "alpha.md", "needle\n")
	mkfile(t, mount, "middle.md", "no match here\nneedle\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "needle")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	wantLines := []string{
		"alpha.md:1:needle",
		"middle.md:2:needle",
		"zebra.md:1:needle",
	}
	gotLines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("got %d lines, want %d:\n%q", len(gotLines), len(wantLines), out)
	}
	for i, want := range wantLines {
		if gotLines[i] != want {
			t.Errorf("line %d: got %q, want %q", i, gotLines[i], want)
		}
	}
}

func TestGrepJSONFormat(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "a.md", "hello\nworld\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "hello", "--format", "json")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	var result struct {
		Matches []struct {
			Path string `json:"path"`
			Line int    `json:"line"`
			Text string `json:"text"`
		} `json:"matches"`
		Truncated  bool   `json:"truncated"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json unmarshal: %v (output=%q)", err, out)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("matches: got %d, want 1", len(result.Matches))
	}
	m := result.Matches[0]
	if m.Path != "a.md" {
		t.Errorf("path: got %q, want %q", m.Path, "a.md")
	}
	if m.Line != 1 {
		t.Errorf("line: got %d, want 1", m.Line)
	}
	if m.Text != "hello" {
		t.Errorf("text: got %q, want %q", m.Text, "hello")
	}
	if result.Truncated {
		t.Errorf("truncated: got true, want false")
	}
	if result.NextCursor != "" {
		t.Errorf("next_cursor: got %q, want empty", result.NextCursor)
	}
}

func TestGrepNoMatchesIsEmpty(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "file.md", "nothing relevant here\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	// Text format: empty output.
	out, errOut, rc := runDispatch(t, "grep", "--pattern", "XYZZY_NOT_PRESENT")
	if rc != ExitOK {
		t.Fatalf("text rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("text stdout: got %q, want empty", out)
	}

	// JSON format: empty matches array, truncated=false.
	out, errOut, rc = runDispatch(t, "grep", "--pattern", "XYZZY_NOT_PRESENT", "--format", "json")
	if rc != ExitOK {
		t.Fatalf("json rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	var result struct {
		Matches   []interface{} `json:"matches"`
		Truncated bool          `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json unmarshal: %v (output=%q)", err, out)
	}
	if len(result.Matches) != 0 {
		t.Errorf("matches: got %d, want 0", len(result.Matches))
	}
	if result.Truncated {
		t.Errorf("truncated: got true, want false")
	}
}

// ---- Filtering tests ----

func TestGrepPathScope(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "root.md", "needle in root\n")
	mkfile(t, mount, "subdir/deep.md", "needle in subdir\n")
	mkfile(t, mount, "other/side.md", "needle in other\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "needle", "--path", "subdir")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if !strings.Contains(out, "subdir/deep.md") {
		t.Errorf("missing subdir/deep.md in output: %q", out)
	}
	if strings.Contains(out, "root.md") {
		t.Errorf("root.md should not appear (outside scope): %q", out)
	}
	if strings.Contains(out, "other/side.md") {
		t.Errorf("other/side.md should not appear (outside scope): %q", out)
	}
}

func TestGrepFileTypeFilter(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "doc.md", "needle md\n")
	mkfile(t, mount, "script.txt", "needle txt\n")
	mkfile(t, mount, "data.json", "needle json\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "needle", "--file-type", "md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if !strings.Contains(out, "doc.md") {
		t.Errorf("doc.md missing from output: %q", out)
	}
	if strings.Contains(out, "script.txt") {
		t.Errorf("script.txt should be filtered out: %q", out)
	}
	if strings.Contains(out, "data.json") {
		t.Errorf("data.json should be filtered out: %q", out)
	}
}

func TestGrepDotfilesSkipped(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, ".mycelium/log.jsonl", "needle in log\n")
	mkfile(t, mount, ".hidden", "needle in hidden\n")
	mkfile(t, mount, "visible.md", "no match here\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "needle")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if strings.Contains(out, ".mycelium") {
		t.Errorf("dotfile dir appeared in output: %q", out)
	}
	if strings.Contains(out, ".hidden") {
		t.Errorf("hidden dotfile appeared in output: %q", out)
	}
	if out != "" {
		t.Errorf("expected empty output (no non-dotfile matches), got %q", out)
	}
}

// ---- Truncation tests ----

func TestGrepLimitTruncates(t *testing.T) {
	mount := t.TempDir()
	// 5 matching lines.
	mkfile(t, mount, "many.md", "match one\nmatch two\nmatch three\nmatch four\nmatch five\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "match", "--limit", "2")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// Expect 2 match lines + 1 truncation line.
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (2 matches + truncation):\n%q", len(lines), out)
	}
	if !strings.Contains(lines[2], "truncated") {
		t.Errorf("last line should be truncation notice, got %q", lines[2])
	}
	if !strings.Contains(lines[2], "2") {
		t.Errorf("truncation notice should include count 2, got %q", lines[2])
	}

	// Verify JSON format also reports truncated=true.
	outJSON, errOutJSON, rcJSON := runDispatch(t, "grep", "--pattern", "match", "--limit", "2", "--format", "json")
	if rcJSON != ExitOK {
		t.Fatalf("json rc: got %d, want %d (stderr=%q)", rcJSON, ExitOK, errOutJSON)
	}
	var result struct {
		Matches   []interface{} `json:"matches"`
		Truncated bool          `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(outJSON), &result); err != nil {
		t.Fatalf("json unmarshal: %v (output=%q)", err, outJSON)
	}
	if len(result.Matches) != 2 {
		t.Errorf("json matches: got %d, want 2", len(result.Matches))
	}
	if !result.Truncated {
		t.Errorf("json truncated: got false, want true")
	}
}

func TestGrepUnderLimitNotTruncated(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "few.md", "match one\nmatch two\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "match", "--limit", "10")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if strings.Contains(out, "truncated") {
		t.Errorf("should not have truncation notice, got %q", out)
	}

	// JSON format.
	outJSON, errOutJSON, rcJSON := runDispatch(t, "grep", "--pattern", "match", "--limit", "10", "--format", "json")
	if rcJSON != ExitOK {
		t.Fatalf("json rc: got %d, want %d (stderr=%q)", rcJSON, ExitOK, errOutJSON)
	}
	var result struct {
		Truncated bool `json:"truncated"`
	}
	if err := json.Unmarshal([]byte(outJSON), &result); err != nil {
		t.Fatalf("json unmarshal: %v (output=%q)", err, outJSON)
	}
	if result.Truncated {
		t.Errorf("json truncated: got true, want false")
	}
}

// ---- Error tests ----

func TestGrepEmptyPatternIsUsageError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "grep", "--pattern", "")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "--pattern is required") {
		t.Errorf("stderr should mention --pattern is required, got %q", errOut)
	}
}

func TestGrepInvalidRegexIsUsageError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "grep", "--pattern", "[invalid", "--regex")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "invalid regex") {
		t.Errorf("stderr should mention invalid regex, got %q", errOut)
	}
}

func TestGrepInvalidFormatIsUsageError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "grep", "--pattern", "foo", "--format", "xml")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "--format must be text or json") {
		t.Errorf("stderr should mention --format must be text or json, got %q", errOut)
	}
}

func TestGrepNonPositiveLimitIsUsageError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	for _, limit := range []string{"0", "-1", "-100"} {
		_, errOut, rc := runDispatch(t, "grep", "--pattern", "foo", "--limit", limit)
		if rc != ExitUsage {
			t.Errorf("limit=%s: rc: got %d, want %d (stderr=%q)", limit, rc, ExitUsage, errOut)
		}
		if !strings.Contains(errOut, "--limit must be positive") {
			t.Errorf("limit=%s: stderr should mention --limit must be positive, got %q", limit, errOut)
		}
	}
}

func TestGrepPathNotFoundIsError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "grep", "--pattern", "foo", "--path", "nonexistent/subdir")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "not found") {
		t.Errorf("stderr should mention not found, got %q", errOut)
	}
}

func TestGrepMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")

	_, errOut, rc := runDispatch(t, "grep", "--pattern", "foo")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

func TestGrepAbsolutePathRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "grep", "--pattern", "foo", "--path", "/etc")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
}

func TestGrepTraversalRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "grep", "--pattern", "foo", "--path", "../escape")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "escapes") {
		t.Errorf("stderr should mention escapes, got %q", errOut)
	}
}

// ---- Cursor stub test ----

func TestGrepCursorAccepted(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "file.md", "hello world\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	// With cursor — should behave exactly the same as without.
	outWithCursor, errOut, rc := runDispatch(t, "grep", "--pattern", "hello", "--cursor", "anything")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	// Without cursor.
	outWithout, _, _ := runDispatch(t, "grep", "--pattern", "hello")

	if outWithCursor != outWithout {
		t.Errorf("cursor flag changed output:\nwith:    %q\nwithout: %q", outWithCursor, outWithout)
	}
}

// ---- Additional edge-case tests ----

func TestGrepJSONEmptyMatchesArray(t *testing.T) {
	// Explicitly verify the JSON output uses [] not null for empty matches.
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "nothing", "--format", "json")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if !strings.Contains(out, `"matches":[]`) {
		t.Errorf("json should contain matches:[], got %q", out)
	}
}

func TestGrepLineNumbersAre1Indexed(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "lines.md", "no match\nno match\nfound it\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "found it")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	// "found it" is on line 3.
	if !strings.Contains(out, "lines.md:3:found it") {
		t.Errorf("expected line 3 match, got %q", out)
	}
}

func TestGrepTextIncludesNoTrailingNewlineInText(t *testing.T) {
	// Text field in output should not have trailing newline from file.
	mount := t.TempDir()
	mkfile(t, mount, "f.md", "match me\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "match me", "--format", "json")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	var result struct {
		Matches []struct {
			Text string `json:"text"`
		} `json:"matches"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matches))
	}
	if result.Matches[0].Text != "match me" {
		t.Errorf("text should be %q without trailing newline, got %q", "match me", result.Matches[0].Text)
	}
}

func TestGrepLimitTruncatesAcrossMultipleFiles(t *testing.T) {
	mount := t.TempDir()
	// 3 files, each with 1 match.
	mkfile(t, mount, "a.md", "needle\n")
	mkfile(t, mount, "b.md", "needle\n")
	mkfile(t, mount, "c.md", "needle\n")
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "grep", "--pattern", "needle", "--limit", "2")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	// Should have 2 match lines + truncation line.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d output lines, want 3:\n%q", len(lines), out)
	}
	if !strings.Contains(lines[2], "truncated") {
		t.Errorf("last line should be truncation notice, got %q", lines[2])
	}
}

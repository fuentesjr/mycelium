package mycelium

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMvHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "moved content\n"
	writeTestFile(t, mount, "src.md", content)

	out, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("stdout: got %q, want empty", out)
	}
	// src should be gone.
	if _, err := os.Stat(filepath.Join(mount, "src.md")); !os.IsNotExist(err) {
		t.Error("src should have been removed")
	}
	// dst should have src's content.
	disk, err := os.ReadFile(filepath.Join(mount, "dst.md"))
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(disk) != content {
		t.Errorf("dst content: got %q, want %q", string(disk), content)
	}
}

func TestMvSrcMissingIsError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "nonexistent.md", "dst.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "not found") {
		t.Errorf("stderr should mention not found, got %q", errOut)
	}
}

func TestMvDstExistsIsConflict(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "source\n")
	dstContent := "original dst\n"
	writeTestFile(t, mount, "dst.md", dstContent)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitConflict {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseDstExistsEnvelope(t, errOut)
	if env.Op != "mv" {
		t.Errorf("envelope op: got %q, want %q", env.Op, "mv")
	}
	if env.Path != "dst.md" {
		t.Errorf("envelope path: got %q, want %q", env.Path, "dst.md")
	}
	if env.CurrentVersion != sha256Hex(dstContent) {
		t.Errorf("envelope current_version: got %q, want %q", env.CurrentVersion, sha256Hex(dstContent))
	}
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent without flag, got %q", *env.CurrentContent)
	}
	if env.ExpectedVersion != "" {
		t.Errorf("expected_version should be absent, got %q", env.ExpectedVersion)
	}
	// Both files must be untouched.
	srcDisk, _ := os.ReadFile(filepath.Join(mount, "src.md"))
	if string(srcDisk) != "source\n" {
		t.Errorf("src should be untouched, got %q", srcDisk)
	}
	dstDisk, _ := os.ReadFile(filepath.Join(mount, "dst.md"))
	if string(dstDisk) != dstContent {
		t.Errorf("dst should be untouched, got %q", dstDisk)
	}
}

func TestMvDstExistsEnvelopeFields(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "source\n")
	dstContent := "existing dst\n"
	writeTestFile(t, mount, "dst.md", dstContent)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseDstExistsEnvelope(t, errOut)
	if env.Error != "destination_exists" {
		t.Errorf("error field: got %q, want %q", env.Error, "destination_exists")
	}
	if env.Op != "mv" {
		t.Errorf("op: got %q, want %q", env.Op, "mv")
	}
	if env.Path != "dst.md" {
		t.Errorf("path: got %q, want %q", env.Path, "dst.md")
	}
	if env.CurrentVersion != sha256Hex(dstContent) {
		t.Errorf("current_version: got %q, want %q", env.CurrentVersion, sha256Hex(dstContent))
	}
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent without flag")
	}
	if env.ExpectedVersion != "" {
		t.Errorf("expected_version should be absent, got %q", env.ExpectedVersion)
	}
}

// parseDstExistsEnvelope parses the first line of stderr as a destination_exists envelope.
func parseDstExistsEnvelope(t *testing.T, stderr string) conflictResult {
	t.Helper()
	line := strings.TrimRight(stderr, "\n")
	if idx := strings.Index(line, "\n"); idx >= 0 {
		line = line[:idx]
	}
	var env conflictResult
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		t.Fatalf("parseDstExistsEnvelope: stderr is not valid JSON: %v\nstderr was: %q", err, stderr)
	}
	if env.Error != "destination_exists" {
		t.Errorf("envelope error field: got %q, want %q", env.Error, "destination_exists")
	}
	return env
}

func TestMvSrcEqualsDst(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "same.md", "content\n")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "same.md", "same.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "same") {
		t.Errorf("stderr should mention same, got %q", errOut)
	}
	// File must still exist and be unchanged.
	disk, _ := os.ReadFile(filepath.Join(mount, "same.md"))
	if string(disk) != "content\n" {
		t.Errorf("file should be untouched, got %q", disk)
	}
}

func TestMvCASMatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "versioned\n"
	writeTestFile(t, mount, "src.md", content)
	ver := sha256Hex(content)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md", "--expected-version", ver)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if _, err := os.Stat(filepath.Join(mount, "dst.md")); err != nil {
		t.Errorf("dst should exist after move: %v", err)
	}
}

func TestMvCASMismatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "original\n"
	writeTestFile(t, mount, "src.md", content)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md", "--expected-version", "sha256:deadbeef")
	if rc != ExitConflict {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Op != "mv" {
		t.Errorf("envelope op: got %q, want %q", env.Op, "mv")
	}
	if env.Path != "src.md" {
		t.Errorf("envelope path: got %q, want %q (must be src, not dst)", env.Path, "src.md")
	}
	if env.CurrentVersion != sha256Hex(content) {
		t.Errorf("envelope current_version: got %q, want %q", env.CurrentVersion, sha256Hex(content))
	}
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent without flag, got %q", *env.CurrentContent)
	}
	// src must be untouched.
	disk, _ := os.ReadFile(filepath.Join(mount, "src.md"))
	if string(disk) != content {
		t.Errorf("src should be untouched, got %q", disk)
	}
	// dst must not exist.
	if _, err := os.Stat(filepath.Join(mount, "dst.md")); !os.IsNotExist(err) {
		t.Error("dst should not exist after CAS mismatch")
	}
}

func TestMvCreatesParentDirs(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "data\n")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "deep/nested/dst.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if _, err := os.Stat(filepath.Join(mount, "deep", "nested", "dst.md")); err != nil {
		t.Errorf("expected nested dst: %v", err)
	}
}

func TestMvMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
}

func TestMvAbsoluteSrcRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "/etc/passwd", "dst.md")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
}

func TestMvTraversalInSrcRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "../escape.md", "dst.md")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "escapes") {
		t.Errorf("stderr should mention escapes, got %q", errOut)
	}
}

func TestMvTraversalInDstRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "x\n")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "../escape.md")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "escapes") {
		t.Errorf("stderr should mention escapes, got %q", errOut)
	}
}

func TestMvLogEntryWrittenOnSuccess(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "log me\n"
	writeTestFile(t, mount, "src.md", content)
	wantVersion := sha256Hex(content)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "mv" {
		t.Errorf("op: got %q, want %q", e.Op, "mv")
	}
	if e.Path != "dst.md" {
		t.Errorf("path: got %q, want %q (should be DST)", e.Path, "dst.md")
	}
	if e.From != "src.md" {
		t.Errorf("from: got %q, want %q", e.From, "src.md")
	}
	if e.Version != wantVersion {
		t.Errorf("version: got %q, want %q", e.Version, wantVersion)
	}
}

func TestMvLogEntryNotWrittenOnFailure(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "x\n")
	writeTestFile(t, mount, "dst.md", "y\n")

	// dst exists — mv must fail without writing a log entry.
	_, _, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc == ExitOK {
		t.Fatal("expected failure when dst exists")
	}

	if logExists(mount) {
		t.Error("log file should not exist after failed mv")
	}
}

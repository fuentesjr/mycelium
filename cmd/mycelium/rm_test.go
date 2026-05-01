package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRmHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "to be removed\n"
	writeTestFile(t, mount, "target.md", content)

	out, errOut, rc := runDispatchWithStdin(t, "", "rm", "target.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("stdout: got %q, want empty", out)
	}
	if _, err := os.Stat(filepath.Join(mount, "target.md")); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

func TestRmFileMissingIsError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "nonexistent.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "not found") {
		t.Errorf("stderr should mention not found, got %q", errOut)
	}
}

func TestRmCASMatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "versioned\n"
	writeTestFile(t, mount, "f.md", content)
	ver := sha256Hex(content)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "f.md", "--expected-version", ver)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if _, err := os.Stat(filepath.Join(mount, "f.md")); !os.IsNotExist(err) {
		t.Error("file should have been deleted on CAS match")
	}
}

func TestRmCASMismatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "original\n"
	writeTestFile(t, mount, "f.md", content)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "f.md", "--expected-version", "sha256:deadbeef")
	if rc != ExitConflict {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Op != "rm" {
		t.Errorf("envelope op: got %q, want %q", env.Op, "rm")
	}
	if env.Path != "f.md" {
		t.Errorf("envelope path: got %q, want %q", env.Path, "f.md")
	}
	if env.CurrentVersion != sha256Hex(content) {
		t.Errorf("envelope current_version: got %q, want %q", env.CurrentVersion, sha256Hex(content))
	}
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent without flag, got %q", *env.CurrentContent)
	}
	// File must be untouched.
	disk, _ := os.ReadFile(filepath.Join(mount, "f.md"))
	if string(disk) != content {
		t.Errorf("file should be untouched on CAS mismatch, got %q", disk)
	}
}

func TestRmMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "f.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
}

func TestRmAbsolutePathRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "/etc/passwd")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
}

func TestRmTraversalRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "../escape.md")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "escapes") {
		t.Errorf("stderr should mention escapes, got %q", errOut)
	}
}

func TestRmLogEntryWrittenOnSuccess(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "log me\n"
	writeTestFile(t, mount, "notes.md", content)
	wantVersion := sha256Hex(content)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "notes.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "rm" {
		t.Errorf("op: got %q, want %q", e.Op, "rm")
	}
	if e.Path != "notes.md" {
		t.Errorf("path: got %q, want %q", e.Path, "notes.md")
	}
	if e.PriorVersion != wantVersion {
		t.Errorf("prior_version: got %q, want %q", e.PriorVersion, wantVersion)
	}
	if e.Version != "" {
		t.Errorf("version should be absent for rm, got %q", e.Version)
	}
}

func TestRmLogEntryNotWrittenOnFailure(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	// File doesn't exist — rm must fail without writing a log entry.
	_, _, rc := runDispatchWithStdin(t, "", "rm", "nonexistent.md")
	if rc == ExitOK {
		t.Fatal("expected failure for missing file")
	}

	if logExists(mount) {
		t.Error("log file should not exist after failed rm")
	}
}

func TestRmIncludeCurrentContentUTF8(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "to be removed\n"
	writeTestFile(t, mount, "f.md", content)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "f.md",
		"--expected-version", "sha256:deadbeef",
		"--include-current-content")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.CurrentContent == nil {
		t.Fatal("current_content should be present for UTF-8 file")
	}
	if *env.CurrentContent != content {
		t.Errorf("current_content: got %q, want %q", *env.CurrentContent, content)
	}
}

func TestRmIncludeCurrentContentBinary(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	abs := filepath.Join(mount, "bin.dat")
	if err := os.WriteFile(abs, []byte{0xff, 0xfe, 0x00}, 0o644); err != nil {
		t.Fatalf("write binary file: %v", err)
	}

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "bin.dat",
		"--expected-version", "sha256:deadbeef",
		"--include-current-content")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent for binary file, got %q", *env.CurrentContent)
	}
}

func TestRmIncludeCurrentContentAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "missing.md",
		"--expected-version", "sha256:deadbeef",
		"--include-current-content")
	// rm returns ExitGenericError when file doesn't exist (not found before CAS check).
	if rc != ExitGenericError {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	_ = errOut
}

// TestRmDirectHelper exercises removeFile directly for coverage of the helper.
func TestRmDirectHelper(t *testing.T) {
	mount := t.TempDir()
	writeTestFile(t, mount, "h.md", "hello\n")

	var errBuf strings.Builder
	ver, rc := removeFile(&errBuf, mount, "h.md", "", false)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errBuf.String())
	}
	if ver != sha256Hex("hello\n") {
		t.Errorf("priorVersion: got %q, want %q", ver, sha256Hex("hello\n"))
	}
	if _, err := os.Stat(filepath.Join(mount, "h.md")); !os.IsNotExist(err) {
		t.Error("file should be gone")
	}
}

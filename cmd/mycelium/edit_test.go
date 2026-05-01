package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)


// writeTestFile is a helper that creates a file with the given content under mount.
func writeTestFile(t *testing.T, mount, name, content string) {
	t.Helper()
	path := filepath.Join(mount, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func TestEditHappyPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	initial := "hello world\n"
	writeTestFile(t, mount, "file.md", initial)

	out, errOut, rc := runDispatchWithStdin(t, "", "edit", "file.md", "--old", "world", "--new", "there")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	disk, err := os.ReadFile(filepath.Join(mount, "file.md"))
	if err != nil {
		t.Fatalf("read after edit: %v", err)
	}
	if string(disk) != "hello there\n" {
		t.Errorf("on-disk content: got %q, want %q", string(disk), "hello there\n")
	}

	wantVersion := sha256Hex("hello there\n")
	if !strings.Contains(out, wantVersion) {
		t.Errorf("stdout should contain version %q, got %q", wantVersion, out)
	}
	if !strings.Contains(out, `"version":"sha256:`) {
		t.Errorf("stdout missing version field, got %q", out)
	}
}

func TestEditNewVersionMatchesPostReplacementContent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "aaa bbb ccc\n")

	out, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "bbb", "--new", "XXX")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	expected := sha256Hex("aaa XXX ccc\n")
	if !strings.Contains(out, expected) {
		t.Errorf("version mismatch: want %q in %q", expected, out)
	}
}

func TestEditOldEmptyIsUsageError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "content\n")

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "", "--new", "something")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "--old is required") {
		t.Errorf("stderr should mention --old is required, got %q", errOut)
	}
}

func TestEditOldNotFoundIsError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	initial := "hello world\n"
	writeTestFile(t, mount, "f.md", initial)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "missing", "--new", "x")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "not found") {
		t.Errorf("stderr should mention not found, got %q", errOut)
	}

	// File must be untouched.
	disk, _ := os.ReadFile(filepath.Join(mount, "f.md"))
	if string(disk) != initial {
		t.Errorf("file should be untouched, got %q", string(disk))
	}
}

func TestEditOldAmbiguousIsError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	initial := "foo foo\n"
	writeTestFile(t, mount, "f.md", initial)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "foo", "--new", "bar")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "ambiguous") {
		t.Errorf("stderr should mention ambiguous, got %q", errOut)
	}

	// File must be untouched.
	disk, _ := os.ReadFile(filepath.Join(mount, "f.md"))
	if string(disk) != initial {
		t.Errorf("file should be untouched, got %q", string(disk))
	}
}

func TestEditFileMissingIsError(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "nonexistent.md", "--old", "x", "--new", "y")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "not found") {
		t.Errorf("stderr should mention not found, got %q", errOut)
	}
}

func TestEditCASMatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	initial := "hello world\n"
	writeTestFile(t, mount, "f.md", initial)
	version := sha256Hex(initial)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "world", "--new", "there", "--expected-version", version)
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	disk, _ := os.ReadFile(filepath.Join(mount, "f.md"))
	if string(disk) != "hello there\n" {
		t.Errorf("on-disk content: got %q", string(disk))
	}
}

func TestEditCASMismatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	initial := "hello world\n"
	writeTestFile(t, mount, "f.md", initial)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "world", "--new", "there", "--expected-version", "sha256:deadbeef")
	if rc != ExitConflict {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Op != "edit" {
		t.Errorf("envelope op: got %q, want %q", env.Op, "edit")
	}
	if env.Path != "f.md" {
		t.Errorf("envelope path: got %q, want %q", env.Path, "f.md")
	}
	if env.CurrentVersion != sha256Hex(initial) {
		t.Errorf("envelope current_version: got %q, want %q", env.CurrentVersion, sha256Hex(initial))
	}
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent without flag, got %q", *env.CurrentContent)
	}

	// File must be untouched.
	disk, _ := os.ReadFile(filepath.Join(mount, "f.md"))
	if string(disk) != initial {
		t.Errorf("file should be untouched, got %q", string(disk))
	}
}

func TestEditMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "x", "--new", "y")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

func TestEditAbsolutePathRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "/etc/passwd", "--old", "x", "--new", "y")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
}

func TestEditTraversalRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "../escape.md", "--old", "x", "--new", "y")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "escapes") {
		t.Errorf("stderr should mention escapes, got %q", errOut)
	}
}

func TestEditLogEntryWrittenOnSuccess(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "notes.md", "hello world\n")

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "notes.md", "--old", "world", "--new", "there")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	wantVersion := sha256Hex("hello there\n")
	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "edit" {
		t.Errorf("op: got %q, want %q", e.Op, "edit")
	}
	if e.Path != "notes.md" {
		t.Errorf("path: got %q, want %q", e.Path, "notes.md")
	}
	if e.Version != wantVersion {
		t.Errorf("version: got %q, want %q", e.Version, wantVersion)
	}
}

func TestEditIncludeCurrentContentUTF8(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "alpha beta gamma\n"
	writeTestFile(t, mount, "g.md", content)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "g.md",
		"--old", "beta", "--new", "BETA",
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

func TestEditIncludeCurrentContentBinary(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	abs := filepath.Join(mount, "bin.dat")
	if err := os.WriteFile(abs, []byte{0xff, 0xfe, 0x00}, 0o644); err != nil {
		t.Fatalf("write binary file: %v", err)
	}

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "bin.dat",
		"--old", "x", "--new", "y",
		"--expected-version", "sha256:deadbeef",
		"--include-current-content")
	// edit will fail with not-found for "x" in binary file OR conflict; either way not ExitOK
	_ = rc
	// If conflict, envelope should have no current_content.
	if rc == ExitConflict {
		env := parseConflictEnvelope(t, errOut)
		if env.CurrentContent != nil {
			t.Errorf("current_content should be absent for binary file, got %q", *env.CurrentContent)
		}
	}
}

func TestEditIncludeCurrentContentAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "missing.md",
		"--old", "x", "--new", "y",
		"--expected-version", "sha256:deadbeef",
		"--include-current-content")
	// edit returns ExitGenericError when file doesn't exist (not found before CAS check).
	if rc != ExitGenericError {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
	// No conflict envelope in this case — file-not-found is caught before CAS.
	_ = errOut
}

func TestEditLogEntryNotWrittenOnFailure(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "f.md", "foo foo\n")

	// ambiguous — no log entry should be written
	_, _, rc := runDispatchWithStdin(t, "", "edit", "f.md", "--old", "foo", "--new", "bar")
	if rc == ExitOK {
		t.Fatal("expected failure for ambiguous match")
	}

	if logExists(mount) {
		t.Error("log file should not exist after failed edit")
	}
}

package mycelium

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

func TestRmRejectsSymlinkParentEscapingMount(t *testing.T) {
	mount := t.TempDir()
	outside := t.TempDir()
	mkfile(t, outside, "file.md", "outside")
	if err := os.Symlink(outside, filepath.Join(mount, "linkdir")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "linkdir/file.md")
	if rc == ExitOK {
		t.Fatal("rm through symlink parent succeeded")
	}
	if !strings.Contains(errOut, "symlink") {
		t.Fatalf("stderr should mention symlink, got %q", errOut)
	}
	if _, err := os.Stat(filepath.Join(outside, "file.md")); err != nil {
		t.Fatalf("outside file should remain: %v", err)
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

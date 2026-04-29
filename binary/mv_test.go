package main

import (
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
	if out != `{"log_status":"ok"}`+"\n" {
		t.Errorf("stdout: got %q, want log_status ok", out)
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
	writeTestFile(t, mount, "dst.md", "original dst\n")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "dst.md")
	if rc != ExitConflict {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	if !strings.Contains(errOut, "destination exists") {
		t.Errorf("stderr should mention destination exists, got %q", errOut)
	}
	// Both files must be untouched.
	srcDisk, _ := os.ReadFile(filepath.Join(mount, "src.md"))
	if string(srcDisk) != "source\n" {
		t.Errorf("src should be untouched, got %q", srcDisk)
	}
	dstDisk, _ := os.ReadFile(filepath.Join(mount, "dst.md"))
	if string(dstDisk) != "original dst\n" {
		t.Errorf("dst should be untouched, got %q", dstDisk)
	}
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

// TestMvDirectHelper exercises moveFile directly for coverage of the helper.
func TestMvDirectHelper(t *testing.T) {
	mount := t.TempDir()
	writeTestFile(t, mount, "a.md", "hello\n")

	var errBuf strings.Builder
	ver, rc := moveFile(&errBuf, mount, "a.md", "b.md", "")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errBuf.String())
	}
	if ver != sha256Hex("hello\n") {
		t.Errorf("version: got %q, want %q", ver, sha256Hex("hello\n"))
	}
	if _, err := os.Stat(filepath.Join(mount, "a.md")); !os.IsNotExist(err) {
		t.Error("src should be gone")
	}
	disk, _ := os.ReadFile(filepath.Join(mount, "b.md"))
	if string(disk) != "hello\n" {
		t.Errorf("dst content: got %q", disk)
	}
}

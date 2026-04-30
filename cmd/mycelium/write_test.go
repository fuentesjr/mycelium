package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sha256Hex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestWriteFileCreatesNewFile(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "first revision\n"
	out, errOut, rc := runDispatchWithStdin(t, content, "write", "memory.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	disk, err := os.ReadFile(filepath.Join(mount, "memory.md"))
	if err != nil {
		t.Fatalf("read after write: %v", err)
	}
	if string(disk) != content {
		t.Errorf("on-disk content: got %q, want %q", string(disk), content)
	}
	wantVersion := sha256Hex(content)
	if !strings.Contains(out, wantVersion) {
		t.Errorf("stdout should contain version %q, got %q", wantVersion, out)
	}
	if !strings.Contains(out, `"log_status":"ok"`) {
		t.Errorf("stdout missing log_status, got %q", out)
	}
}

func TestWriteFileCreatesParentDir(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "deep/nested/file.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}
	if _, err := os.Stat(filepath.Join(mount, "deep", "nested", "file.md")); err != nil {
		t.Errorf("expected nested file: %v", err)
	}
}

func TestWriteFileCASMatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	initial := "v1"
	if _, _, rc := runDispatchWithStdin(t, initial, "write", "m.md"); rc != ExitOK {
		t.Fatal("setup write failed")
	}
	expected := sha256Hex(initial)
	_, errOut, rc := runDispatchWithStdin(t, "v2", "write", "m.md", "--expected-version", expected)
	if rc != ExitOK {
		t.Fatalf("rc: got %d (stderr=%q)", rc, errOut)
	}
	disk, _ := os.ReadFile(filepath.Join(mount, "m.md"))
	if string(disk) != "v2" {
		t.Errorf("expected v2 on disk, got %q", disk)
	}
}

func TestWriteFileCASMismatch(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	if _, _, rc := runDispatchWithStdin(t, "v1", "write", "m.md"); rc != ExitOK {
		t.Fatal("setup write failed")
	}
	_, errOut, rc := runDispatchWithStdin(t, "v2", "write", "m.md", "--expected-version", "sha256:deadbeef")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	disk, _ := os.ReadFile(filepath.Join(mount, "m.md"))
	if string(disk) != "v1" {
		t.Errorf("file should not have been overwritten on conflict, got %q", disk)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.Op != "write" {
		t.Errorf("envelope op: got %q, want %q", env.Op, "write")
	}
	if env.Path != "m.md" {
		t.Errorf("envelope path: got %q, want %q", env.Path, "m.md")
	}
	if env.CurrentVersion != sha256Hex("v1") {
		t.Errorf("envelope current_version: got %q, want %q", env.CurrentVersion, sha256Hex("v1"))
	}
	if env.ExpectedVersion != "sha256:deadbeef" {
		t.Errorf("envelope expected_version: got %q, want %q", env.ExpectedVersion, "sha256:deadbeef")
	}
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent without flag, got %q", *env.CurrentContent)
	}
}

func TestWriteFileCASOnAbsentFileConflicts(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "new.md", "--expected-version", "sha256:deadbeef")
	if rc != ExitConflict {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.CurrentVersion != "sha256:absent" {
		t.Errorf("envelope current_version: got %q, want sha256:absent", env.CurrentVersion)
	}
}

func TestWriteFileExpectedVersionMustHavePrefix(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "m.md", "--expected-version", "garbage")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
}

func TestWriteFileMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")
	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "m.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitGenericError, errOut)
	}
}

func TestWriteLogsOnSuccess(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "logged content\n"
	_, errOut, rc := runDispatchWithStdin(t, content, "write", "notes.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	wantVersion := sha256Hex(content)
	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Op != "write" {
		t.Errorf("op: got %q, want %q", e.Op, "write")
	}
	if e.Path != "notes.md" {
		t.Errorf("path: got %q, want %q", e.Path, "notes.md")
	}
	if e.Version != wantVersion {
		t.Errorf("version: got %q, want %q", e.Version, wantVersion)
	}
}

func TestWriteIncludeCurrentContentUTF8(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	content := "hello, world\n"
	writeTestFile(t, mount, "f.md", content)

	_, errOut, rc := runDispatchWithStdin(t, "new content", "write", "f.md",
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

func TestWriteIncludeCurrentContentBinary(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	abs := filepath.Join(mount, "bin.dat")
	if err := os.WriteFile(abs, []byte{0xff, 0xfe, 0x00}, 0o644); err != nil {
		t.Fatalf("write binary file: %v", err)
	}
	binVer := sha256Hex(string([]byte{0xff, 0xfe, 0x00}))

	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "bin.dat",
		"--expected-version", "sha256:deadbeef",
		"--include-current-content")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent for binary file, got %q", *env.CurrentContent)
	}
	if env.CurrentVersion != binVer {
		t.Errorf("current_version: got %q, want %q", env.CurrentVersion, binVer)
	}
}

func TestWriteIncludeCurrentContentAbsent(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "missing.md",
		"--expected-version", "sha256:deadbeef",
		"--include-current-content")
	if rc != ExitConflict {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitConflict, errOut)
	}
	env := parseConflictEnvelope(t, errOut)
	if env.CurrentVersion != "sha256:absent" {
		t.Errorf("current_version: got %q, want sha256:absent", env.CurrentVersion)
	}
	if env.CurrentContent != nil {
		t.Errorf("current_content should be absent when file doesn't exist, got %q", *env.CurrentContent)
	}
}

func TestWriteFilePathTraversalRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "../escape.md")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitUsage, errOut)
	}
	if !strings.Contains(errOut, "escapes") {
		t.Errorf("stderr should mention escape, got %q", errOut)
	}
}

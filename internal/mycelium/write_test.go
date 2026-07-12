package mycelium

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
	if !strings.Contains(out, `"version":"sha256:`) {
		t.Errorf("stdout missing version field, got %q", out)
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

func TestWriteRejectsSymlinkParentEscapingMount(t *testing.T) {
	mount := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(mount, "linkdir")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "new", "write", "linkdir/file.md")
	if rc == ExitOK {
		t.Fatal("write through symlink parent succeeded")
	}
	if !strings.Contains(errOut, "symlink") {
		t.Fatalf("stderr should mention symlink, got %q", errOut)
	}
	if _, err := os.Stat(filepath.Join(outside, "file.md")); !os.IsNotExist(err) {
		t.Fatalf("outside file should not be created (err=%v)", err)
	}
}

func TestWriteRejectsSymlinkLeaf(t *testing.T) {
	mount := t.TempDir()
	outside := t.TempDir()
	mkfile(t, outside, "file.md", "outside")
	if err := os.Symlink(filepath.Join(outside, "file.md"), filepath.Join(mount, "file.md")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "new", "write", "file.md")
	if rc == ExitOK {
		t.Fatal("write through symlink leaf succeeded")
	}
	if !strings.Contains(errOut, "symlink") {
		t.Fatalf("stderr should mention symlink, got %q", errOut)
	}
	disk, err := os.ReadFile(filepath.Join(outside, "file.md"))
	if err != nil || string(disk) != "outside" {
		t.Fatalf("outside file changed: content=%q err=%v", disk, err)
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

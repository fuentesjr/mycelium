package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileHappyPath(t *testing.T) {
	mount := t.TempDir()
	contents := "hello world\n"
	if err := os.WriteFile(filepath.Join(mount, "memory.md"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "read", "memory.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != contents {
		t.Errorf("stdout: got %q, want %q", out, contents)
	}
}

func TestReadFileNested(t *testing.T) {
	mount := t.TempDir()
	dir := filepath.Join(mount, "notes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "today.md"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, _, rc := runDispatch(t, "read", "notes/today.md")
	if rc != ExitOK || out != "nested" {
		t.Errorf("got rc=%d out=%q, want rc=0 out=%q", rc, out, "nested")
	}
}

func TestReadFileMissing(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatch(t, "read", "nope.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "not found") {
		t.Errorf("stderr should mention 'not found', got %q", errOut)
	}
}

func TestReadFileMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")
	_, errOut, rc := runDispatch(t, "read", "memory.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

func TestReadFilePathTraversalRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatch(t, "read", "../escape.md")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d", rc, ExitUsage)
	}
	if !strings.Contains(errOut, "escapes mount root") {
		t.Errorf("stderr should mention escape, got %q", errOut)
	}
}

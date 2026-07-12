package mycelium

import (
	"encoding/json"
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

func TestReadFileJSONFormat(t *testing.T) {
	mount := t.TempDir()
	content := "hello json\n"
	if err := os.WriteFile(filepath.Join(mount, "memory.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "read", "memory.md", "--format", "json")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}

	var env readEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("stdout is not JSON: %v; out=%q", err, out)
	}
	if env.Path != "memory.md" {
		t.Errorf("path: got %q, want memory.md", env.Path)
	}
	if env.Content != content {
		t.Errorf("content: got %q, want %q", env.Content, content)
	}
	if env.Version != sha256Hex(content) {
		t.Errorf("version: got %q, want %q", env.Version, sha256Hex(content))
	}
}

func TestReadRejectsSymlinkLeafEscapingMount(t *testing.T) {
	mount := t.TempDir()
	outside := t.TempDir()
	mkfile(t, outside, "secret.md", "outside secret")
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(mount, "link.md")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "read", "link.md")
	if rc == ExitOK {
		t.Fatalf("rc: got OK, want failure (out=%q)", out)
	}
	if !strings.Contains(errOut, "symlink") {
		t.Fatalf("stderr should mention symlink, got %q", errOut)
	}
	if strings.Contains(out, "outside secret") {
		t.Fatalf("outside content leaked: %q", out)
	}
}

func TestReadRejectsSymlinkParent(t *testing.T) {
	mount := t.TempDir()
	outside := t.TempDir()
	mkfile(t, outside, "secret.md", "outside secret")
	if err := os.Symlink(outside, filepath.Join(mount, "linkdir")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)

	out, errOut, rc := runDispatch(t, "read", "linkdir/secret.md")
	if rc == ExitOK {
		t.Fatalf("rc: got OK, want failure (out=%q)", out)
	}
	if !strings.Contains(errOut, "symlink") {
		t.Fatalf("stderr should mention symlink, got %q", errOut)
	}
}

func TestReadFileInvalidFormat(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatch(t, "read", "memory.md", "--format", "xml")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d", rc, ExitUsage)
	}
	if !strings.Contains(errOut, "--format") {
		t.Errorf("stderr should mention --format, got %q", errOut)
	}
}

func TestReadFileJSONRejectsInvalidUTF8(t *testing.T) {
	mount := t.TempDir()
	if err := os.WriteFile(filepath.Join(mount, "binary.dat"), []byte{0xff, 0xfe, 0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatch(t, "read", "binary.dat", "--format", "json")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "UTF-8") {
		t.Errorf("stderr should mention UTF-8, got %q", errOut)
	}

	out, _, rc := runDispatch(t, "read", "binary.dat")
	if rc != ExitOK {
		t.Fatalf("text read rc: got %d, want %d", rc, ExitOK)
	}
	if []byte(out)[0] != 0xff {
		t.Errorf("text read should preserve raw bytes, got %v", []byte(out))
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

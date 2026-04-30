package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkfile creates a file (and any necessary parent directories) under dir with
// the given content.
func mkfile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkfile mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("mkfile write: %v", err)
	}
}

// ---- listFiles unit tests ----

func TestListFilesEmptyMount(t *testing.T) {
	mount := t.TempDir()
	files, err := listFiles(mount, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty, got %v", files)
	}
}

func TestListFilesSingleTopLevel(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "memory.md", "hi")

	files, err := listFiles(mount, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "memory.md" {
		t.Errorf("got %v, want [memory.md]", files)
	}
}

func TestListFilesMultipleTopLevelAlphabetical(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "zebra.md", "z")
	mkfile(t, mount, "apple.md", "a")
	mkfile(t, mount, "mango.md", "m")

	files, err := listFiles(mount, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"apple.md", "mango.md", "zebra.md"}
	if len(files) != len(want) {
		t.Fatalf("got %v, want %v", files, want)
	}
	for i, f := range files {
		if f != want[i] {
			t.Errorf("index %d: got %q, want %q", i, f, want[i])
		}
	}
}

func TestListFilesNonRecursiveOmitsSubdirFiles(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "top.md", "top")
	mkfile(t, mount, "notes/nested.md", "nested")

	files, err := listFiles(mount, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "top.md" {
		t.Errorf("got %v, want [top.md]", files)
	}
}

func TestListFilesRecursiveIncludesNested(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "root.md", "r")
	mkfile(t, mount, "notes/today.md", "today")
	mkfile(t, mount, "notes/archive/old.md", "old")

	files, err := listFiles(mount, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"notes/archive/old.md", "notes/today.md", "root.md"}
	if len(files) != len(want) {
		t.Fatalf("got %v, want %v", files, want)
	}
	for i, f := range files {
		if f != want[i] {
			t.Errorf("index %d: got %q, want %q", i, f, want[i])
		}
	}
}

func TestListFilesSkipsDotMyceliumDir(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "visible.md", "v")
	mkfile(t, mount, ".mycelium/log.jsonl", "{}")

	for _, recursive := range []bool{false, true} {
		files, err := listFiles(mount, recursive)
		if err != nil {
			t.Fatalf("recursive=%v unexpected error: %v", recursive, err)
		}
		for _, f := range files {
			if strings.Contains(f, ".mycelium") {
				t.Errorf("recursive=%v: .mycelium appeared in output: %v", recursive, files)
			}
		}
		if len(files) != 1 || files[0] != "visible.md" {
			t.Errorf("recursive=%v: got %v, want [visible.md]", recursive, files)
		}
	}
}

func TestListFilesSkipsTopLevelDotfiles(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "real.md", "r")
	mkfile(t, mount, ".DS_Store", "mac")

	files, err := listFiles(mount, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range files {
		if strings.HasPrefix(f, ".") {
			t.Errorf("dotfile appeared in output: %q", f)
		}
	}
	if len(files) != 1 || files[0] != "real.md" {
		t.Errorf("got %v, want [real.md]", files)
	}
}

func TestListFilesForwardSlashPaths(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "a/b/c.md", "deep")

	files, err := listFiles(mount, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %v", files)
	}
	if strings.Contains(files[0], "\\") {
		t.Errorf("path contains backslash: %q", files[0])
	}
	if files[0] != "a/b/c.md" {
		t.Errorf("got %q, want %q", files[0], "a/b/c.md")
	}
}

// ---- globMatches unit tests ----

func TestGlobMatchesBasenamePattern(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "notes/today.md", "t")
	mkfile(t, mount, "notes/today.txt", "t")
	mkfile(t, mount, "archive/old.md", "o")
	mkfile(t, mount, ".hidden.md", "h")

	matches, err := globMatches(mount, "*.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"archive/old.md", "notes/today.md"}
	if len(matches) != len(want) {
		t.Fatalf("got %v, want %v", matches, want)
	}
	for i, m := range matches {
		if m != want[i] {
			t.Errorf("index %d: got %q, want %q", i, m, want[i])
		}
	}
}

func TestGlobMatchesFullPathPattern(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "notes/today.md", "n")
	mkfile(t, mount, "other/today.md", "o")
	mkfile(t, mount, "notes/readme.txt", "r")

	matches, err := globMatches(mount, "notes/*.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 || matches[0] != "notes/today.md" {
		t.Errorf("got %v, want [notes/today.md]", matches)
	}
}

func TestGlobMatchesNoMatches(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "readme.txt", "r")

	matches, err := globMatches(mount, "*.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %v", matches)
	}
}

func TestGlobMatchesInvalidPattern(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "file.md", "f")

	_, err := globMatches(mount, "[")
	if err == nil {
		t.Fatal("expected error for invalid pattern, got nil")
	}
}

func TestGlobMatchesSkipsDotfiles(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "real.md", "r")
	mkfile(t, mount, ".hidden.md", "h")
	mkfile(t, mount, ".mycelium/log.jsonl", "{}")

	matches, err := globMatches(mount, "*.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, m := range matches {
		if strings.HasPrefix(filepath.Base(m), ".") || strings.Contains(m, ".mycelium") {
			t.Errorf("dotfile appeared in glob results: %q", m)
		}
	}
	if len(matches) != 1 || matches[0] != "real.md" {
		t.Errorf("got %v, want [real.md]", matches)
	}
}

// ---- E2E tests through runDispatch ----

func TestLsE2EEmptyMount(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "ls")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("stdout: got %q, want empty", out)
	}
}

func TestLsE2ETopLevelFiles(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "b.md", "b")
	mkfile(t, mount, "a.md", "a")
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "ls")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "a.md\nb.md\n" {
		t.Errorf("stdout: got %q, want %q", out, "a.md\nb.md\n")
	}
}

func TestLsE2ERecursive(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "root.md", "r")
	mkfile(t, mount, "sub/deep.md", "d")
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "ls", "--recursive")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "root.md\nsub/deep.md\n" {
		t.Errorf("stdout: got %q, want %q", out, "root.md\nsub/deep.md\n")
	}
}

func TestLsE2ESkipsDotMycelium(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "real.md", "r")
	mkfile(t, mount, ".mycelium/log.jsonl", "{}")
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "ls", "--recursive")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if strings.Contains(out, ".mycelium") {
		t.Errorf("output contains .mycelium: %q", out)
	}
	if out != "real.md\n" {
		t.Errorf("stdout: got %q, want %q", out, "real.md\n")
	}
}

func TestLsE2EMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")
	_, errOut, rc := runDispatch(t, "ls")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

func TestGlobE2EBasenamePattern(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "notes/today.md", "t")
	mkfile(t, mount, "readme.txt", "r")
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "glob", "*.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "notes/today.md\n" {
		t.Errorf("stdout: got %q, want %q", out, "notes/today.md\n")
	}
}

func TestGlobE2EFullPathPattern(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "notes/today.md", "n")
	mkfile(t, mount, "other/today.md", "o")
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "glob", "notes/*.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "notes/today.md\n" {
		t.Errorf("stdout: got %q, want %q", out, "notes/today.md\n")
	}
}

func TestGlobE2ENoMatches(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "readme.txt", "r")
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "glob", "*.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("stdout: got %q, want empty", out)
	}
}

func TestGlobE2EInvalidPattern(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "file.md", "f")
	t.Setenv("MYCELIUM_MOUNT", mount)
	_, errOut, rc := runDispatch(t, "glob", "[")
	if rc != ExitUsage {
		t.Errorf("rc: got %d, want %d", rc, ExitUsage)
	}
	if !strings.Contains(errOut, "invalid pattern") {
		t.Errorf("stderr should mention 'invalid pattern', got %q", errOut)
	}
}

func TestGlobE2ESkipsDotfiles(t *testing.T) {
	mount := t.TempDir()
	mkfile(t, mount, "real.md", "r")
	mkfile(t, mount, ".hidden.md", "h")
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, _, rc := runDispatch(t, "glob", "*.md")
	if rc != ExitOK {
		t.Fatalf("rc: got %d, want %d", rc, ExitOK)
	}
	if strings.Contains(out, ".hidden") {
		t.Errorf("output contains hidden dotfile: %q", out)
	}
	if out != "real.md\n" {
		t.Errorf("stdout: got %q, want %q", out, "real.md\n")
	}
}

func TestGlobE2EMountUnset(t *testing.T) {
	t.Setenv("MYCELIUM_MOUNT", "")
	_, errOut, rc := runDispatch(t, "glob", "*.md")
	if rc != ExitGenericError {
		t.Errorf("rc: got %d, want %d", rc, ExitGenericError)
	}
	if !strings.Contains(errOut, "MYCELIUM_MOUNT") {
		t.Errorf("stderr should mention MYCELIUM_MOUNT, got %q", errOut)
	}
}

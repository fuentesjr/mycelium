package mycelium

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUnderMount(t *testing.T) {
	mount := t.TempDir()
	tests := []struct {
		name      string
		mount     string
		requested string
		wantErr   error
		wantSub   string
	}{
		{"simple relative", mount, "memory.md", nil, "memory.md"},
		{"nested", mount, "notes/today.md", nil, filepath.Join("notes", "today.md")},
		{"dot prefix collapses", mount, "./memory.md", nil, "memory.md"},
		{"empty path", mount, "", ErrPathEmpty, ""},
		{"empty mount", "", "memory.md", ErrMountUnset, ""},
		{"absolute path", mount, "/etc/passwd", ErrPathAbsolute, ""},
		{"escapes via parent", mount, "../escape.md", ErrPathEscapesRoot, ""},
		{"escapes via nested parent", mount, "a/../../escape.md", ErrPathEscapesRoot, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveUnderMount(tt.mount, tt.requested)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err: got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := filepath.Join(tt.mount, tt.wantSub)
			if got != want {
				t.Errorf("path: got %q, want %q", got, want)
			}
		})
	}
}

func TestRejectSymlinkComponents(t *testing.T) {
	mount := t.TempDir()
	outside := t.TempDir()
	mkfile(t, mount, "dir/file.md", "ok")
	mkfile(t, outside, "secret.md", "secret")
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(mount, "leaf.md")); err != nil {
		t.Fatalf("symlink leaf: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(mount, "linkdir")); err != nil {
		t.Fatalf("symlink dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want error
	}{
		{"regular path", filepath.Join(mount, "dir", "file.md"), nil},
		{"missing suffix after regular parent", filepath.Join(mount, "dir", "new.md"), nil},
		{"symlink leaf", filepath.Join(mount, "leaf.md"), ErrPathSymlink},
		{"symlink parent", filepath.Join(mount, "linkdir", "secret.md"), ErrPathSymlink},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rejectSymlinkComponents(mount, tt.path)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err: got %v, want %v", err, tt.want)
			}
		})
	}
}

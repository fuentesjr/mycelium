package main

import (
	"errors"
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

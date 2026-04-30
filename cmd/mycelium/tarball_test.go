package main

import (
	"archive/tar"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestStoreRoundTripsThroughTar pins Phase 1 acceptance #8: the store is
// plain files plus JSONL by design. We tar a populated mount, untar to a
// fresh directory, and inspect it using only the Go standard library —
// os.ReadFile as a proxy for cat, regexp as a proxy for grep, and
// encoding/json to confirm the activity log is well-formed JSONL. Pinning
// this contract as a test means a future change can't quietly introduce a
// binary or proprietary container format without a failing test.
func TestStoreRoundTripsThroughTar(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	t.Setenv("MYCELIUM_AGENT_ID", "tarball-test-agent")

	writes := []struct {
		path    string
		content string
	}{
		{"notes/alpha.md", "first thoughts on cryptographic agility"},
		{"notes/beta.md", "follow-up: consider hybrid kex modes"},
		{"plans/sprint-1.md", "milestone: ship pi.dev integration"},
	}
	for _, w := range writes {
		_, errOut, rc := runDispatchWithStdin(t, w.content, "write", w.path)
		if rc != ExitOK {
			t.Fatalf("seed write %s: rc=%d stderr=%q", w.path, rc, errOut)
		}
	}
	_, errOut, rc := runDispatch(t, "log", "context_signal", "--payload-json", `{"note":"benchmark anchor"}`)
	if rc != ExitOK {
		t.Fatalf("seed log: rc=%d stderr=%q", rc, errOut)
	}

	tarPath := filepath.Join(t.TempDir(), "store.tar")
	if err := tarDir(mount, tarPath); err != nil {
		t.Fatalf("tar: %v", err)
	}

	extracted := t.TempDir()
	if err := untar(tarPath, extracted); err != nil {
		t.Fatalf("untar: %v", err)
	}

	// Plain-file read (cat proxy).
	alpha, err := os.ReadFile(filepath.Join(extracted, "notes", "alpha.md"))
	if err != nil {
		t.Fatalf("read alpha: %v", err)
	}
	if string(alpha) != writes[0].content {
		t.Errorf("alpha content: got %q, want %q", string(alpha), writes[0].content)
	}

	// Pattern search (grep proxy).
	beta, err := os.ReadFile(filepath.Join(extracted, "notes", "beta.md"))
	if err != nil {
		t.Fatalf("read beta: %v", err)
	}
	if !regexp.MustCompile(`(?i)hybrid`).Match(beta) {
		t.Errorf("regexp match on beta: no match in %q", beta)
	}

	// Activity log: exactly one daily file, well-formed JSONL with the
	// expected entry count and required fields per line.
	matches, err := filepath.Glob(filepath.Join(extracted, "_activity", "*", "*", "*", "tarball-test-agent.jsonl"))
	if err != nil {
		t.Fatalf("glob activity: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("activity log files: got %d, want 1 (matches=%v)", len(matches), matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read activity: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	wantEntries := len(writes) + 1 // 3 writes + 1 explicit log
	if len(lines) != wantEntries {
		t.Errorf("activity entries: got %d, want %d", len(lines), wantEntries)
	}
	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d: invalid JSON: %v (line=%q)", i, err, line)
			continue
		}
		for _, key := range []string{"ts", "op", "agent_id"} {
			if _, ok := entry[key]; !ok {
				t.Errorf("line %d: missing key %q (line=%q)", i, key, line)
			}
		}
	}
}

// tarDir writes every entry under src to a tar archive at dst, with names
// stored relative to src. It walks in lexical order (filepath.Walk's
// guarantee), so directories are emitted before their contents.
func tarDir(src, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	defer tw.Close()
	return filepath.Walk(src, func(p string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rf, err := os.Open(p)
		if err != nil {
			return err
		}
		defer rf.Close()
		_, err = io.Copy(tw, rf)
		return err
	})
}

// untar extracts a tar archive at src into dst. Refuses entries that
// would resolve outside dst (defensive against path traversal).
func untar(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	tr := tar.NewReader(f)
	cleanDst := filepath.Clean(dst) + string(os.PathSeparator)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(hdr.Name))
		if !strings.HasPrefix(target, cleanDst) {
			return os.ErrInvalid
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			wf, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(wf, tr); err != nil {
				_ = wf.Close()
				return err
			}
			if err := wf.Close(); err != nil {
				return err
			}
		}
	}
}

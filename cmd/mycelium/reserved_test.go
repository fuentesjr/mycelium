package main

import (
	"strings"
	"testing"
)

// TestReservedPrefixWriteRejected verifies that writing to _activity/foo.md returns 65.
func TestReservedPrefixWriteRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "_activity/foo.md")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixWriteAnyPrefix verifies that any _-prefixed root is rejected.
func TestReservedPrefixWriteAnyPrefix(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "_custom/foo.md")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixUnderscoreOnly verifies that _/foo.md is rejected.
func TestReservedPrefixUnderscoreOnly(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "_/foo.md")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixBareUnderscore verifies that bare _ is rejected.
func TestReservedPrefixBareUnderscore(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "_")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixEditRejected verifies edit is also blocked.
func TestReservedPrefixEditRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "edit", "_activity/foo.md", "--old", "x", "--new", "y")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixRmRejected verifies rm is also blocked.
func TestReservedPrefixRmRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "rm", "_activity/foo.md")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixMvSrcRejected verifies mv src is also blocked.
func TestReservedPrefixMvSrcRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "_activity/foo.md", "dst.md")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixMvDstRejected verifies mv dst is also blocked.
func TestReservedPrefixMvDstRejected(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	writeTestFile(t, mount, "src.md", "data\n")

	_, errOut, rc := runDispatchWithStdin(t, "", "mv", "src.md", "_activity/dst.md")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
	if !strings.Contains(errOut, "reserved") {
		t.Errorf("stderr should mention reserved, got %q", errOut)
	}
}

// TestReservedPrefixOnlyRootSegment verifies that nested _-prefixed subdirs are allowed.
func TestReservedPrefixOnlyRootSegment(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "content\n", "write", "notes/_sub/foo.md")
	if rc != ExitOK {
		t.Errorf("rc: got %d, want %d (stderr=%q) — nested _-prefix should be allowed", rc, ExitOK, errOut)
	}
}

// TestReservedPrefixLogsAllowed verifies that logs/ (not _-prefixed) is freely writable.
func TestReservedPrefixLogsAllowed(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "content\n", "write", "logs/foo.md")
	if rc != ExitOK {
		t.Errorf("rc: got %d, want %d (stderr=%q) — logs/ should be freely agent-writable", rc, ExitOK, errOut)
	}
}

// TestReservedPrefixDoesNotBlockInternalLogging verifies that the auto-log on
// mutation still works correctly (i.e. internal writes to _activity/ continue
// to succeed) by asserting that a successful write produces a log entry.
func TestReservedPrefixDoesNotBlockInternalLogging(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "hello\n", "write", "notes.md")
	if rc != ExitOK {
		t.Fatalf("write failed: rc=%d stderr=%q", rc, errOut)
	}

	// Internal log should have been written to _activity/ despite the agent
	// write being to notes.md (not _-prefixed).
	entries := readLogLines(t, mount)
	if len(entries) != 1 {
		t.Fatalf("expected 1 activity entry from auto-log, got %d", len(entries))
	}
	if entries[0].Op != "write" {
		t.Errorf("expected op=write, got %q", entries[0].Op)
	}
}

// TestReservedPrefixTestReservedPath exercises the synthetic _test_reserved/ prefix.
func TestReservedPrefixTestReservedPath(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)

	_, errOut, rc := runDispatchWithStdin(t, "x", "write", "_test_reserved/data.md")
	if rc != ExitReservedPrefix {
		t.Errorf("rc: got %d, want %d (stderr=%q)", rc, ExitReservedPrefix, errOut)
	}
}

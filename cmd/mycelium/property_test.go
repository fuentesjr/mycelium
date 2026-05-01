package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

// ---- custom types with quick.Generator implementations ----

// reservedPath is a relative path whose first segment starts with '_'.
// Guaranteed to pass the shape rules but trigger ExitReservedPrefix.
type reservedPath string

func (reservedPath) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(reservedPath(genPath(r, true)))
}

// validPath is a relative path whose first segment does NOT start with '_'.
// Guaranteed to pass shape rules and not trigger ExitReservedPrefix.
type validPath string

func (validPath) Generate(r *rand.Rand, _ int) reflect.Value {
	return reflect.ValueOf(validPath(genPath(r, false)))
}

// smallContent is an ASCII-printable string of 1-200 chars.
type smallContent string

func (smallContent) Generate(r *rand.Rand, _ int) reflect.Value {
	const printable = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !@#$%^&*()_+-=[]{}|;:,.<>?"
	n := 1 + r.Intn(200)
	b := make([]byte, n)
	for i := range b {
		b[i] = printable[r.Intn(len(printable))]
	}
	return reflect.ValueOf(smallContent(b))
}

// genSegment returns a non-empty alphanumeric ASCII segment of length 1-8.
func genSegment(r *rand.Rand) string {
	const alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	n := 1 + r.Intn(8)
	b := make([]byte, n)
	for i := range b {
		b[i] = alpha[r.Intn(len(alpha))]
	}
	return string(b)
}

// genPath builds a path with 1-4 segments. If reserved is true the first
// segment is prefixed with '_'; otherwise it starts with an alphanumeric char.
// Additional segments (0-3) are appended separated by '/'.
func genPath(r *rand.Rand, reserved bool) string {
	first := genSegment(r)
	if reserved {
		first = "_" + first
	}
	numExtra := r.Intn(4) // 0-3 additional segments
	parts := make([]string, 1, 1+numExtra)
	parts[0] = first
	for i := 0; i < numExtra; i++ {
		parts = append(parts, genSegment(r))
	}
	return strings.Join(parts, "/")
}

// ---- write stdout envelope ----

type writeResult struct {
	Version string `json:"version"`
}

// captureWriteVersion runs "write path" with stdin content and returns the
// version token. Fatals on unexpected exit code.
func captureWriteVersion(t *testing.T, path, content string) string {
	t.Helper()
	out, errOut, rc := runDispatchWithStdin(t, content, "write", path)
	if rc != ExitOK {
		t.Fatalf("write %q: rc=%d stderr=%q", path, rc, errOut)
	}
	var wr writeResult
	if err := json.Unmarshal([]byte(strings.TrimRight(out, "\n")), &wr); err != nil {
		t.Fatalf("write %q: parse stdout JSON: %v\nstdout=%q", path, err, out)
	}
	if wr.Version == "" {
		t.Fatalf("write %q: empty version in stdout=%q", path, out)
	}
	return wr.Version
}

// ---- Property 1: _-prefix reservation ----

// TestProperty_ReservedPrefixAlwaysRejected checks that for any path whose
// first segment starts with '_', every agent-facing mutating op exits 65 and
// stderr contains "reserved".
func TestProperty_ReservedPrefixAlwaysRejected(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(1)),
	}

	check := func(p reservedPath) bool {
		path := string(p)
		mount := t.TempDir()
		t.Setenv("MYCELIUM_MOUNT", mount)

		passed := true

		t.Run(fmt.Sprintf("reserved/%q", path), func(t *testing.T) {
			// write
			_, stderr, rc := runDispatchWithStdin(t, "content", "write", path)
			if rc != ExitReservedPrefix {
				t.Errorf("write %q: rc=%d want %d", path, rc, ExitReservedPrefix)
				passed = false
			}
			if !strings.Contains(stderr, "reserved") {
				t.Errorf("write %q: stderr=%q missing 'reserved'", path, stderr)
				passed = false
			}

			// edit
			_, stderr, rc = runDispatchWithStdin(t, "", "edit", path, "--old", "x", "--new", "y")
			if rc != ExitReservedPrefix {
				t.Errorf("edit %q: rc=%d want %d", path, rc, ExitReservedPrefix)
				passed = false
			}
			if !strings.Contains(stderr, "reserved") {
				t.Errorf("edit %q: stderr=%q missing 'reserved'", path, stderr)
				passed = false
			}

			// rm
			_, stderr, rc = runDispatchWithStdin(t, "", "rm", path)
			if rc != ExitReservedPrefix {
				t.Errorf("rm %q: rc=%d want %d", path, rc, ExitReservedPrefix)
				passed = false
			}
			if !strings.Contains(stderr, "reserved") {
				t.Errorf("rm %q: stderr=%q missing 'reserved'", path, stderr)
				passed = false
			}

			// mv with reserved src
			_, stderr, rc = runDispatchWithStdin(t, "", "mv", path, "dst.md")
			if rc != ExitReservedPrefix {
				t.Errorf("mv-src %q: rc=%d want %d", path, rc, ExitReservedPrefix)
				passed = false
			}
			if !strings.Contains(stderr, "reserved") {
				t.Errorf("mv-src %q: stderr=%q missing 'reserved'", path, stderr)
				passed = false
			}

			// mv with reserved dst — seed a valid src first
			writeTestFile(t, mount, "mv_prop_src.md", "seed")
			_, stderr, rc = runDispatchWithStdin(t, "", "mv", "mv_prop_src.md", path)
			if rc != ExitReservedPrefix {
				t.Errorf("mv-dst %q: rc=%d want %d", path, rc, ExitReservedPrefix)
				passed = false
			}
			if !strings.Contains(stderr, "reserved") {
				t.Errorf("mv-dst %q: stderr=%q missing 'reserved'", path, stderr)
				passed = false
			}
		})

		return passed
	}

	if err := quick.Check(check, cfg); err != nil {
		t.Error(err)
	}
}

// TestProperty_ValidPathNeverReturns65 checks that for any path whose first
// segment does NOT start with '_', every agent-facing mutating op never exits
// ExitReservedPrefix=65.
func TestProperty_ValidPathNeverReturns65(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(1)),
	}

	check := func(p validPath) bool {
		path := string(p)
		mount := t.TempDir()
		t.Setenv("MYCELIUM_MOUNT", mount)

		passed := true

		t.Run(fmt.Sprintf("valid/%q", path), func(t *testing.T) {
			// write — seed the file so subsequent ops have something to work with
			_, _, rc := runDispatchWithStdin(t, "seed content", "write", path)
			if rc == ExitReservedPrefix {
				t.Errorf("write %q: unexpected ExitReservedPrefix", path)
				passed = false
			}

			// edit on the pre-seeded file
			_, _, rc = runDispatchWithStdin(t, "", "edit", path, "--old", "seed", "--new", "replaced")
			if rc == ExitReservedPrefix {
				t.Errorf("edit %q: unexpected ExitReservedPrefix", path)
				passed = false
			}

			// rm
			_, _, rc = runDispatchWithStdin(t, "", "rm", path)
			if rc == ExitReservedPrefix {
				t.Errorf("rm %q: unexpected ExitReservedPrefix", path)
				passed = false
			}

			// mv-src: seed a different file and move it to path
			srcPath := "mv_valid_src.md"
			writeTestFile(t, mount, srcPath, "mv seed")
			_, _, rc = runDispatchWithStdin(t, "", "mv", srcPath, path)
			if rc == ExitReservedPrefix {
				t.Errorf("mv-src into %q: unexpected ExitReservedPrefix", path)
				passed = false
			}

			// mv-dst: seed path (may have been removed above) and move to a fixed dst
			writeTestFile(t, mount, path, "mv dst seed")
			_, _, rc = runDispatchWithStdin(t, "", "mv", path, "mv_valid_dst.md")
			if rc == ExitReservedPrefix {
				t.Errorf("mv-dst from %q: unexpected ExitReservedPrefix", path)
				passed = false
			}
		})

		return passed
	}

	if err := quick.Check(check, cfg); err != nil {
		t.Error(err)
	}
}

// ---- Property 2: CAS optimistic concurrency ----

// TestProperty_CASConflictEnvelope verifies the full CAS conflict contract for
// the write operation across a random space of (path, content_v0, content_v1)
// triples.
func TestProperty_CASConflictEnvelope(t *testing.T) {
	cfg := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(1)),
	}

	check := func(p validPath, v0 smallContent, v1 smallContent) bool {
		// We need two distinct content strings; if the generator produces equal
		// ones, append a distinguishing byte so the versions are guaranteed to
		// differ, keeping the test well-defined.
		c0 := string(v0)
		c1 := string(v1)
		if c0 == c1 {
			c1 += "X"
		}

		path := string(p)
		mount := t.TempDir()
		t.Setenv("MYCELIUM_MOUNT", mount)

		passed := true

		t.Run(fmt.Sprintf("cas/%q", path), func(t *testing.T) {
			// Step 1: write v0, capture token.
			tok0 := captureWriteVersion(t, path, c0)

			// Step 2: write v1 unconditionally, capture token.
			tok1 := captureWriteVersion(t, path, c1)

			// Step 3: write with stale expected-version (tok0). Must conflict.
			_, stderr, rc := runDispatchWithStdin(t, "anything", "write", path,
				"--expected-version", tok0)
			if rc != ExitConflict {
				t.Errorf("path=%q: stale write rc=%d want %d", path, rc, ExitConflict)
				passed = false
				return
			}

			// Step 4: parse the conflict envelope.
			env := parseConflictEnvelope(t, stderr)

			// Step 5: current_version must equal tok1.
			if env.CurrentVersion != tok1 {
				t.Errorf("path=%q: current_version=%q want %q", path, env.CurrentVersion, tok1)
				passed = false
			}

			// Step 6: expected_version must equal tok0.
			if env.ExpectedVersion != tok0 {
				t.Errorf("path=%q: expected_version=%q want %q", path, env.ExpectedVersion, tok0)
				passed = false
			}

			// Step 7: op must be "write".
			if env.Op != "write" {
				t.Errorf("path=%q: op=%q want \"write\"", path, env.Op)
				passed = false
			}
		})

		return passed
	}

	if err := quick.Check(check, cfg); err != nil {
		t.Error(err)
	}
}

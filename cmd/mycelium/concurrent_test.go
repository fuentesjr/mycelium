package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestMain re-execs this test binary as the mycelium dispatcher when
// MYCELIUM_TEST_BINARY_MODE=1. This lets concurrent tests spawn real
// sibling processes hitting the same mount, which is the only way to
// validate file-locked CAS semantics across process boundaries. Without
// the env var TestMain runs the test suite normally.
func TestMain(m *testing.M) {
	if os.Getenv("MYCELIUM_TEST_BINARY_MODE") == "1" {
		os.Exit(dispatch(os.Stdin, os.Stdout, os.Stderr, os.Args[1:]))
	}
	os.Exit(m.Run())
}

type childResult struct {
	idx    int
	stdout string
	stderr string
	code   int
	err    error
}

// runMyceliumChild spawns this test binary in dispatcher mode (see
// TestMain) and returns its outcome. Safe to call from goroutines:
// returns errors instead of failing the test directly.
func runMyceliumChild(mount, stdin string, args ...string) (stdout, stderr string, code int, err error) {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(),
		"MYCELIUM_TEST_BINARY_MODE=1",
		"MYCELIUM_MOUNT="+mount,
	)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return stdout, stderr, exitErr.ExitCode(), nil
		}
		return stdout, stderr, -1, runErr
	}
	return stdout, stderr, 0, nil
}

// TestConcurrentCASExclusivity validates Phase 1 acceptance #2 and #7:
// N sibling processes racing the same CAS write must produce exactly
// one winner; the rest must observe the conflict and exit 64. The
// final on-disk content must match the winner's payload — no silent
// loss, no partial overwrite.
func TestConcurrentCASExclusivity(t *testing.T) {
	mount := t.TempDir()
	initial := "v1"
	if _, _, rc, err := runMyceliumChild(mount, initial, "write", "shared.md"); err != nil || rc != ExitOK {
		t.Fatalf("seed write: rc=%d err=%v", rc, err)
	}
	expected := sha256Hex(initial)

	const N = 10
	results := make(chan childResult, N)
	contents := make([]string, N)
	for i := 0; i < N; i++ {
		contents[i] = fmt.Sprintf("writer-%d-content", i)
		go func(idx int) {
			stdout, stderr, code, err := runMyceliumChild(
				mount, contents[idx],
				"write", "shared.md", "--expected-version", expected,
			)
			results <- childResult{idx, stdout, stderr, code, err}
		}(i)
	}

	var winners []int
	for i := 0; i < N; i++ {
		r := <-results
		if r.err != nil {
			t.Errorf("writer %d: child run error: %v", r.idx, r.err)
			continue
		}
		switch r.code {
		case ExitOK:
			winners = append(winners, r.idx)
		case ExitConflict:
			// expected
		default:
			t.Errorf("writer %d: unexpected exit code %d (stdout=%q stderr=%q)",
				r.idx, r.code, r.stdout, r.stderr)
		}
	}

	if len(winners) != 1 {
		t.Fatalf("expected exactly 1 winner, got %d (winners=%v)", len(winners), winners)
	}

	disk, err := os.ReadFile(filepath.Join(mount, "shared.md"))
	if err != nil {
		t.Fatalf("read final: %v", err)
	}
	want := contents[winners[0]]
	if string(disk) != want {
		t.Errorf("final content: got %q, want %q (winner=%d)",
			string(disk), want, winners[0])
	}
}

// TestConcurrentUnconditionalAtomicity validates that under contention
// from N sibling processes doing unconditional writes (no CAS), the
// final file content is exactly one of the inputs — never a torn or
// partial mix. Atomicity is guaranteed by atomicWrite's temp-then-rename;
// the lock additionally serializes the rename ordering.
func TestConcurrentUnconditionalAtomicity(t *testing.T) {
	mount := t.TempDir()
	const N = 10
	contents := make([]string, N)
	for i := 0; i < N; i++ {
		contents[i] = strings.Repeat(fmt.Sprintf("p%02d-", i), 50+i*10)
	}

	results := make(chan childResult, N)
	for i := 0; i < N; i++ {
		go func(idx int) {
			stdout, stderr, code, err := runMyceliumChild(
				mount, contents[idx], "write", "shared.md",
			)
			results <- childResult{idx, stdout, stderr, code, err}
		}(i)
	}
	for i := 0; i < N; i++ {
		r := <-results
		if r.err != nil {
			t.Errorf("writer %d: %v", r.idx, r.err)
			continue
		}
		if r.code != ExitOK {
			t.Errorf("writer %d: rc=%d (stderr=%q)", r.idx, r.code, r.stderr)
		}
	}

	disk, err := os.ReadFile(filepath.Join(mount, "shared.md"))
	if err != nil {
		t.Fatalf("read final: %v", err)
	}
	final := string(disk)
	for _, c := range contents {
		if final == c {
			return
		}
	}
	t.Errorf("final content matches no input (len=%d); torn write or interleave",
		len(final))
}

// TestConcurrentCASAndEditExclusivity exercises a mixed race: half the
// processes do CAS write, half do edit. Exactly one op must win; every
// loser must surface a non-silent failure. Edits can lose two ways
// (ExitConflict if the version check catches the change, or
// ExitGenericError if a winning write already replaced the marker so
// the old-string lookup fails) — both are valid non-silent outcomes.
func TestConcurrentCASAndEditExclusivity(t *testing.T) {
	mount := t.TempDir()
	initial := "marker-XYZ"
	if _, _, rc, err := runMyceliumChild(mount, initial, "write", "shared.md"); err != nil || rc != ExitOK {
		t.Fatalf("seed write: rc=%d err=%v", rc, err)
	}
	expected := sha256Hex(initial)

	const N = 10
	type opSpec struct {
		idx     int
		isEdit  bool
		content string
	}
	specs := make([]opSpec, N)
	for i := 0; i < N; i++ {
		specs[i] = opSpec{
			idx:     i,
			isEdit:  i%2 == 0,
			content: fmt.Sprintf("op-%d-payload", i),
		}
	}

	results := make(chan childResult, N)
	var wg sync.WaitGroup
	for _, s := range specs {
		wg.Add(1)
		go func(s opSpec) {
			defer wg.Done()
			var stdout, stderr string
			var code int
			var err error
			if s.isEdit {
				stdout, stderr, code, err = runMyceliumChild(mount, "",
					"edit", "shared.md",
					"--old", "XYZ", "--new", s.content,
					"--expected-version", expected,
				)
			} else {
				stdout, stderr, code, err = runMyceliumChild(mount, s.content,
					"write", "shared.md", "--expected-version", expected,
				)
			}
			results <- childResult{s.idx, stdout, stderr, code, err}
		}(s)
	}
	wg.Wait()
	close(results)

	successes := 0
	for r := range results {
		if r.err != nil {
			t.Errorf("op %d: %v", r.idx, r.err)
			continue
		}
		switch r.code {
		case ExitOK:
			successes++
		case ExitConflict, ExitGenericError:
			// non-silent loser outcomes
		default:
			t.Errorf("op %d: unexpected exit %d (stderr=%q)", r.idx, r.code, r.stderr)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly 1 success, got %d", successes)
	}
}

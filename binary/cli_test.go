package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// conflictResult is the parsed form of the structured JSON conflict envelope.
type conflictResult struct {
	Error           string `json:"error"`
	Op              string `json:"op"`
	Path            string `json:"path"`
	CurrentVersion  string `json:"current_version"`
	ExpectedVersion string `json:"expected_version"`
	CurrentContent  *string `json:"current_content"`
}

// parseConflictEnvelope parses the first line of stderr as a conflict envelope.
// It fatals if the line is not valid JSON or the "error" field is not "conflict".
func parseConflictEnvelope(t *testing.T, stderr string) conflictResult {
	t.Helper()
	line := strings.TrimRight(stderr, "\n")
	// Take only the first line in case there are extra diagnostics.
	if idx := strings.Index(line, "\n"); idx >= 0 {
		line = line[:idx]
	}
	var env conflictResult
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		t.Fatalf("parseConflictEnvelope: stderr is not valid JSON: %v\nstderr was: %q", err, stderr)
	}
	if env.Error != "conflict" {
		t.Errorf("envelope error field: got %q, want %q", env.Error, "conflict")
	}
	return env
}

func runDispatch(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	return runDispatchWithStdin(t, "", args...)
}

func runDispatchWithStdin(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := dispatch(strings.NewReader(stdin), &out, &errOut, args)
	return out.String(), errOut.String(), code
}

func TestGrepSubcommandShapeParity(t *testing.T) {
	// grep is a real implementation; mount must be set.
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "grep", "--pattern", "foo")
	if rc != ExitOK {
		t.Errorf("exit code: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	if out != "" {
		t.Errorf("stdout: got %q, want empty (no matches)", out)
	}
}

func TestDispatchErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
		wantRC  int
	}{
		{"no args", []string{}, "usage: mycelium", ExitUsage},
		{"unknown", []string{"nope"}, "unknown subcommand", ExitUsage},
		{"read missing path", []string{"read"}, "PATH required", ExitUsage},
		{"write missing path", []string{"write"}, "PATH required", ExitUsage},
		{"edit missing path", []string{"edit"}, "PATH required", ExitUsage},
		{"glob missing pattern", []string{"glob"}, "PATTERN required", ExitUsage},
		{"rm missing path", []string{"rm"}, "PATH required", ExitUsage},
		{"mv missing args", []string{"mv", "only-one"}, "SRC and DST required", ExitUsage},
		{"log missing op", []string{"log"}, "OP required", ExitUsage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errOut, rc := runDispatch(t, tt.args...)
			if rc != tt.wantRC {
				t.Errorf("exit code: got %d, want %d", rc, tt.wantRC)
			}
			if !strings.Contains(errOut, tt.wantErr) {
				t.Errorf("stderr: got %q, want substring %q", errOut, tt.wantErr)
			}
		})
	}
}

// TestLogSubcommandShapeParity covers the log subcommand's output shape,
// which needs a real mount and is therefore separated from the table above.
func TestLogSubcommandShapeParity(t *testing.T) {
	mount := t.TempDir()
	t.Setenv("MYCELIUM_MOUNT", mount)
	out, errOut, rc := runDispatch(t, "log", "context_signal", "--payload-json", "{}")
	if rc != ExitOK {
		t.Fatalf("exit code: got %d, want %d (stderr=%q)", rc, ExitOK, errOut)
	}
	want := `{"log_status":"ok"}` + "\n"
	if out != want {
		t.Errorf("stdout: got %q, want %q", out, want)
	}
}

func TestDispatchListsSubcommands(t *testing.T) {
	_, errOut, _ := runDispatch(t, "nope")
	for _, sc := range subcommands {
		if !strings.Contains(errOut, sc.name) {
			t.Errorf("stderr missing subcommand %q in listing: %q", sc.name, errOut)
		}
	}
}

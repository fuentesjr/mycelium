package main

import (
	"bytes"
	"strings"
	"testing"
)

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

func TestStubbedSubcommandShapeParity(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantOut string
		wantRC  int
	}{
		{"edit", []string{"edit", "foo", "--old", "a", "--new", "b"}, `{"version":"sha256:stubbed","log_status":"ok"}` + "\n", ExitOK},
		{"grep", []string{"grep", "--pattern", "foo"}, "", ExitOK},
		{"rm", []string{"rm", "foo"}, `{"log_status":"ok"}` + "\n", ExitOK},
		{"mv", []string{"mv", "a", "b"}, `{"log_status":"ok"}` + "\n", ExitOK},
		// log is now a real implementation; mount must be set.
		// The test sets MYCELIUM_MOUNT via t.Setenv inside the subtest below.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, rc := runDispatch(t, tt.args...)
			if rc != tt.wantRC {
				t.Errorf("exit code: got %d, want %d (stderr=%q)", rc, tt.wantRC, errOut)
			}
			if out != tt.wantOut {
				t.Errorf("stdout: got %q, want %q", out, tt.wantOut)
			}
		})
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

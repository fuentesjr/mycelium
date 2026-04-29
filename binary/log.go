package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LogEntry is the on-disk record appended to <mount>/.mycelium/log.jsonl.
type LogEntry struct {
	TS        string          `json:"ts"`
	AgentID   string          `json:"agent_id,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Op        string          `json:"op"`
	Path      string          `json:"path,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// appendLog writes one JSON line to <mount>/.mycelium/log.jsonl and prints
// {"log_status":"ok"} to out on success.  now is injected for testability.
func appendLog(
	in io.Reader,
	out, errOut io.Writer,
	id Identity,
	op, path, payloadJSON string,
	fromStdin bool,
	now time.Time,
) int {
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium log: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	// Resolve payload.
	var payload json.RawMessage
	if payloadJSON != "" {
		if !json.Valid([]byte(payloadJSON)) {
			fmt.Fprintln(errOut, "mycelium log: --payload-json is not valid JSON")
			return ExitUsage
		}
		payload = json.RawMessage(payloadJSON)
	} else if fromStdin {
		raw, err := io.ReadAll(in)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium log: read stdin: %v\n", err)
			return ExitGenericError
		}
		if !json.Valid(raw) {
			fmt.Fprintln(errOut, "mycelium log: stdin is not valid JSON")
			return ExitUsage
		}
		payload = json.RawMessage(raw)
	}

	entry := LogEntry{
		TS:        now.UTC().Format(time.RFC3339Nano),
		AgentID:   id.AgentID,
		SessionID: id.SessionID,
		Op:        op,
		Path:      path,
		Payload:   payload,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium log: marshal entry: %v\n", err)
		return ExitGenericError
	}
	line = append(line, '\n')

	logDir := filepath.Join(id.Mount, ".mycelium")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(errOut, "mycelium log: mkdir: %v\n", err)
		return ExitGenericError
	}

	logPath := filepath.Join(logDir, "log.jsonl")
	// TODO: relies on POSIX O_APPEND PIPE_BUF atomicity for small writes; no flock for now.
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium log: open log file: %v\n", err)
		return ExitGenericError
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		fmt.Fprintf(errOut, "mycelium log: write log file: %v\n", err)
		return ExitGenericError
	}

	fmt.Fprint(out, stubLogResponse)
	return ExitOK
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogEntry is the on-disk record appended to _activity/YYYY/MM/DD/<agent_id>.jsonl.
type LogEntry struct {
	TS           string `json:"ts"`
	AgentID      string `json:"agent_id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	Op           string `json:"op"`
	Path         string `json:"path,omitempty"`
	Version      string `json:"version,omitempty"`
	PriorVersion string `json:"prior_version,omitempty"`
	From         string `json:"from,omitempty"`
	SignalPath   string `json:"signal_path,omitempty"`
}

// MutationLog carries the typed fields for a mutation log entry.
type MutationLog struct {
	Op           string
	Path         string
	Version      string
	PriorVersion string
	From         string
}

// logMutation writes a metadata entry to _activity/. Returns "ok" if the
// write succeeded, "missing" if it failed. Stderr warning is still emitted
// on failure. Mutations have already happened by the time we log, so
// failure here is non-fatal.
func logMutation(errOut io.Writer, id Identity, m MutationLog) string {
	var capErr bytes.Buffer
	rc := appendActivity(io.Discard, &capErr, id, LogEntry{
		Op:           m.Op,
		Path:         m.Path,
		Version:      m.Version,
		PriorVersion: m.PriorVersion,
		From:         m.From,
	}, time.Now())
	if rc != ExitOK {
		msg := capErr.String()
		fmt.Fprintf(errOut, "mycelium %s: log entry write failed: %s", m.Op, msg)
		return "missing"
	}
	return "ok"
}

// activityLogPath returns the path to the agent's daily activity log file.
// If agentID is empty, uses "unspecified.jsonl".
func activityLogPath(mount string, agentID string, now time.Time) string {
	fileName := agentID
	if fileName == "" {
		fileName = "unspecified"
	}
	utc := now.UTC()
	return filepath.Join(
		mount,
		"_activity",
		fmt.Sprintf("%04d", utc.Year()),
		fmt.Sprintf("%02d", int(utc.Month())),
		fmt.Sprintf("%02d", utc.Day()),
		fileName+".jsonl",
	)
}

// appendActivity writes one JSON line to the agent's daily _activity file.
// now is injected for testability. Returns ExitOK on success.
func appendActivity(out, errOut io.Writer, id Identity, entry LogEntry, now time.Time) int {
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium log: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	entry.TS = now.UTC().Format(time.RFC3339Nano)
	entry.AgentID = id.AgentID
	entry.SessionID = id.SessionID

	line, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium log: marshal entry: %v\n", err)
		return ExitGenericError
	}
	line = append(line, '\n')

	logPath := activityLogPath(id.Mount, id.AgentID, now)
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(errOut, "mycelium log: mkdir: %v\n", err)
		return ExitGenericError
	}

	// Relies on POSIX O_APPEND PIPE_BUF atomicity for small writes; no flock for now.
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

	if out != nil && out != io.Discard {
		fmt.Fprint(out, stubLogResponse)
	}
	return ExitOK
}

// logsPayloadPath returns the path for storing an agent-supplied payload.
// Format: <mount>/logs/YYYY/MM/DD/<agent_id>/<HHMMSS>.<nanos>-<op>.json
func logsPayloadPath(mount string, agentID string, op string, now time.Time) string {
	agent := agentID
	if agent == "" {
		agent = "unspecified"
	}
	utc := now.UTC()
	filename := fmt.Sprintf(
		"%02d%02d%02d.%09d-%s.json",
		utc.Hour(), utc.Minute(), utc.Second(),
		utc.Nanosecond(),
		op,
	)
	return filepath.Join(
		mount,
		"logs",
		fmt.Sprintf("%04d", utc.Year()),
		fmt.Sprintf("%02d", int(utc.Month())),
		fmt.Sprintf("%02d", utc.Day()),
		agent,
		filename,
	)
}

// relForwardSlash converts an absolute payload path to a forward-slash
// relative path from the mount root.
func relForwardSlash(mount, abs string) string {
	rel, err := filepath.Rel(mount, abs)
	if err != nil {
		return abs
	}
	return strings.ReplaceAll(rel, string(filepath.Separator), "/")
}

// appendLog handles `mycelium log <op>`. It routes agent-supplied payloads
// to logs/ and writes metadata to _activity/. now is injected for testability.
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

	// Reject op values containing path-unsafe characters.
	if strings.ContainsAny(op, "/\\") {
		fmt.Fprintf(errOut, "mycelium log: op %q contains path-unsafe characters\n", op)
		return ExitUsage
	}

	// Resolve payload bytes.
	var payloadBytes []byte
	hasPayload := false
	if payloadJSON != "" {
		if !json.Valid([]byte(payloadJSON)) {
			fmt.Fprintln(errOut, "mycelium log: --payload-json is not valid JSON")
			return ExitUsage
		}
		payloadBytes = []byte(payloadJSON)
		hasPayload = true
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
		payloadBytes = raw
		hasPayload = true
	}

	var signalPath string

	if hasPayload {
		// Write payload to logs/ tree.
		payloadDest := logsPayloadPath(id.Mount, id.AgentID, op, now)
		if err := atomicWrite(payloadDest, payloadBytes); err != nil {
			fmt.Fprintf(errOut, "mycelium log: write payload: %v\n", err)
			return ExitGenericError
		}
		signalPath = relForwardSlash(id.Mount, payloadDest)
	}

	// Write metadata entry to _activity/.
	entry := LogEntry{
		Op:         op,
		Path:       path,
		SignalPath: signalPath,
	}
	rc := appendActivity(io.Discard, errOut, id, entry, now)
	if rc != ExitOK {
		if hasPayload {
			// Payload is preserved; only index entry is missing.
			fmt.Fprint(out, `{"log_status":"missing"}`+"\n")
			return ExitOK
		}
		// No payload — the whole point was to log; treat as failure.
		return rc
	}

	fmt.Fprint(out, stubLogResponse)
	return ExitOK
}

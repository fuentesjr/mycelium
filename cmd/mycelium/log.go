package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// LogEntry is the on-disk record appended to _activity/YYYY/MM/DD/<agent_id>.jsonl.
type LogEntry struct {
	TS           string          `json:"ts"`
	AgentID      string          `json:"agent_id,omitempty"`
	SessionID    string          `json:"session_id,omitempty"`
	TxID         string          `json:"tx_id,omitempty"`
	Op           string          `json:"op"`
	Path         string          `json:"path,omitempty"`
	Version      string          `json:"version,omitempty"`
	PriorVersion string          `json:"prior_version,omitempty"`
	From         string          `json:"from,omitempty"`
	Recovered    bool            `json:"recovered,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Rationale    string          `json:"rationale,omitempty"`
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
//
// Deprecated: CLI content mutations use _tx-backed transactional logging.
func logMutation(errOut io.Writer, id Identity, m MutationLog) string {
	var capErr bytes.Buffer
	rc := appendActivity(&capErr, id, LogEntry{
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

func completeLogEntry(id Identity, entry LogEntry, now time.Time) LogEntry {
	entry.TS = now.UTC().Format(time.RFC3339Nano)
	entry.AgentID = id.AgentID
	entry.SessionID = id.SessionID
	return entry
}

// appendActivity writes one JSON line to the agent's daily _activity file.
// now is injected for testability. Returns ExitOK on success.
func appendActivity(errOut io.Writer, id Identity, entry LogEntry, now time.Time) int {
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium log: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	entry = completeLogEntry(id, entry, now)
	if err := appendActivityEntryDurable(id.Mount, entry); err != nil {
		fmt.Fprintf(errOut, "mycelium log: %v\n", err)
		return ExitGenericError
	}
	return ExitOK
}

// appendActivityEntryDurable appends an already-complete LogEntry exactly as
// supplied. Unlike appendActivity, it does not rewrite ts/agent/session fields;
// recovery uses this to replay a prepared activity entry from _tx/.
func appendActivityEntryDurable(mount string, entry LogEntry) error {
	when, err := time.Parse(time.RFC3339Nano, entry.TS)
	if err != nil {
		return fmt.Errorf("parse activity timestamp: %w", err)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	line = append(line, '\n')
	return appendActivityLineDurable(mount, entry.AgentID, when, line)
}

// appendActivityLineDurable appends a pre-marshaled JSONL line and fsyncs the
// file and containing directory so callers can treat success as durable.
func appendActivityLineDurable(mount, agentID string, when time.Time, line []byte) error {
	logPath := activityLogPath(mount, agentID, when)
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	writeErr := error(nil)
	if n, err := f.Write(line); err != nil {
		writeErr = fmt.Errorf("write log file: %w", err)
	} else if n != len(line) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		if err := f.Sync(); err != nil {
			writeErr = fmt.Errorf("sync log file: %w", err)
		}
	}
	if err := f.Close(); writeErr == nil && err != nil {
		writeErr = fmt.Errorf("close log file: %w", err)
	}
	if writeErr != nil {
		return writeErr
	}
	if err := syncDirAncestors(logDir, mount); err != nil {
		return fmt.Errorf("sync log dirs: %w", err)
	}
	return nil
}

// appendLog handles `mycelium log <op>`. It inlines agent-supplied payloads
// directly on the activity entry as the `payload` field. now is injected for testability.
func appendLog(
	in io.Reader,
	errOut io.Writer,
	id Identity,
	op, path, payloadJSON string,
	fromStdin bool,
	rationale string,
	now time.Time,
) int {
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium log: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	// Resolve payload bytes.
	var payloadBytes []byte
	if payloadJSON != "" {
		if !json.Valid([]byte(payloadJSON)) {
			fmt.Fprintln(errOut, "mycelium log: --payload-json is not valid JSON")
			return ExitUsage
		}
		payloadBytes = []byte(payloadJSON)
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
	}

	release, err := acquireMountLock(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium log: acquire lock: %v\n", err)
		return ExitGenericError
	}
	defer release()

	if rc := recoverPendingTransactions(errOut, id); rc != ExitOK {
		return rc
	}

	// Build entry with payload inlined.
	entry := LogEntry{
		Op:        op,
		Path:      path,
		Payload:   json.RawMessage(payloadBytes),
		Rationale: rationale,
	}
	return appendActivity(errOut, id, entry, now)
}

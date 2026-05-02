package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// kindRegex is the allowed pattern for agent-supplied kind names.
var kindRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const (
	maxKindLen       = 64
	maxRationaleSize = 64 * 1024 // 64 KiB
	maxTargetSize    = 4 * 1024  // 4 KiB
)

// EvolveEntry holds the evolve-specific fields stored in the activity log.
// Fields are marshaled flat onto the LogEntry (not nested under payload).
type EvolveEntry struct {
	// LogEntry common fields are set by appendActivity; these are the extra ones.
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Target         string `json:"target,omitempty"`
	Supersedes     string `json:"supersedes,omitempty"`
	KindDefinition string `json:"kind_definition,omitempty"`
	Rationale      string `json:"rationale"`
}

// evolveLogEntry is the flat on-disk representation we parse when scanning
// the activity log for prior evolve events.
type evolveLogEntry struct {
	TS             string `json:"ts"`
	AgentID        string `json:"agent_id"`
	SessionID      string `json:"session_id"`
	Op             string `json:"op"`
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Target         string `json:"target"`
	Supersedes     string `json:"supersedes"`
	KindDefinition string `json:"kind_definition"`
	Rationale      string `json:"rationale"`
}

// loadEvolveEntries walks all _activity/**/*.jsonl files under mount, parses
// every line, and returns all evolve-op entries in file-encounter order.
// The caller is responsible for holding the mount lock if concurrent mutation
// is a concern.
func loadEvolveEntries(mount string) ([]evolveLogEntry, error) {
	pattern := filepath.Join(mount, "_activity", "*", "*", "*", "*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob activity: %w", err)
	}
	sort.Strings(matches) // chronological order by directory path

	var out []evolveLogEntry
	for _, path := range matches {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 256*1024), 256*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			var e evolveLogEntry
			if err := json.Unmarshal([]byte(line), &e); err != nil {
				// Skip lines that don't parse rather than failing the whole scan.
				continue
			}
			if e.Op == "evolve" {
				out = append(out, e)
			}
		}
		_ = f.Close()
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("scan %s: %w", path, err)
		}
	}
	return out, nil
}

// supersededIDs returns the set of ULID ids that are referenced as superseded
// by any entry in the list.
func supersededIDs(entries []evolveLogEntry) map[string]bool {
	out := make(map[string]bool)
	for _, e := range entries {
		if e.Supersedes != "" {
			out[e.Supersedes] = true
		}
	}
	return out
}

// findByID returns the first evolve entry whose ID matches id, or (zero, false).
func findByID(entries []evolveLogEntry, id string) (evolveLogEntry, bool) {
	for _, e := range entries {
		if e.ID == id {
			return e, true
		}
	}
	return evolveLogEntry{}, false
}

// latestActiveForKindTarget returns the most recent non-superseded evolve entry
// that matches (kind, target), or (zero, false) if none.
func latestActiveForKindTarget(entries []evolveLogEntry, kind, target string) (evolveLogEntry, bool) {
	sup := supersededIDs(entries)
	// Walk in reverse (last seen = most recent for same file-order).
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Kind == kind && e.Target == target && !sup[e.ID] {
			return e, true
		}
	}
	return evolveLogEntry{}, false
}

// kindHasBeenUsed reports whether the given kind has ever appeared in the log
// (any entry, superseded or not, builtin or agent-introduced).
func kindHasBeenUsed(entries []evolveLogEntry, kind string) bool {
	for _, e := range entries {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

// latestKindDefinitionID returns the ID of the most recent active
// _kind_definition event targeting kindName, or "" if none.
func latestKindDefinitionID(entries []evolveLogEntry, kindName string) string {
	sup := supersededIDs(entries)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Kind == reservedKindDefinition && e.Target == kindName && !sup[e.ID] {
			return e.ID
		}
	}
	return ""
}

// validateKind validates agent-supplied kind names per ADR rules.
func validateKind(kind string, errOut io.Writer) int {
	if kind == "" {
		fmt.Fprintln(errOut, "mycelium evolve: kind is required")
		return ExitReservedPrefix
	}
	if len(kind) > maxKindLen {
		fmt.Fprintf(errOut, "mycelium evolve: kind exceeds %d characters\n", maxKindLen)
		return ExitReservedPrefix
	}
	if !kindRegex.MatchString(kind) {
		fmt.Fprintf(errOut, "mycelium evolve: kind %q must match ^[a-zA-Z0-9_-]+$\n", kind)
		return ExitReservedPrefix
	}
	if strings.HasPrefix(kind, "_") {
		fmt.Fprintf(errOut, "mycelium evolve: kind %q: '_'-prefixed kinds are reserved for binary use\n", kind)
		return ExitReservedPrefix
	}
	return ExitOK
}

// appendEvolveEntry marshals an EvolveEntry and appends it to the activity log.
// It uses a flat JSON merge: the EvolveEntry fields are merged into the LogEntry
// JSON so they appear at the top level rather than nested under "payload".
func appendEvolveEntry(errOut io.Writer, id Identity, e EvolveEntry, now time.Time) int {
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium evolve: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	type flatEntry struct {
		TS             string `json:"ts"`
		AgentID        string `json:"agent_id,omitempty"`
		SessionID      string `json:"session_id,omitempty"`
		Op             string `json:"op"`
		ID             string `json:"id"`
		Kind           string `json:"kind"`
		Target         string `json:"target,omitempty"`
		Supersedes     string `json:"supersedes,omitempty"`
		KindDefinition string `json:"kind_definition,omitempty"`
		Rationale      string `json:"rationale"`
	}

	entry := flatEntry{
		TS:             now.UTC().Format(time.RFC3339Nano),
		AgentID:        id.AgentID,
		SessionID:      id.SessionID,
		Op:             "evolve",
		ID:             e.ID,
		Kind:           e.Kind,
		Target:         e.Target,
		Supersedes:     e.Supersedes,
		KindDefinition: e.KindDefinition,
		Rationale:      e.Rationale,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium evolve: marshal entry: %v\n", err)
		return ExitGenericError
	}
	line = append(line, '\n')

	logPath := activityLogPath(id.Mount, id.AgentID, now)
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(errOut, "mycelium evolve: mkdir: %v\n", err)
		return ExitGenericError
	}

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium evolve: open log file: %v\n", err)
		return ExitGenericError
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		fmt.Fprintf(errOut, "mycelium evolve: write log file: %v\n", err)
		return ExitGenericError
	}
	return ExitOK
}

// runEvolve is the handler for `mycelium evolve <kind> [flags]`.
func runEvolve(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("evolve", flag.ContinueOnError)
	fs.SetOutput(errOut)
	targetFlag := fs.String("target", "", "opaque agent-chosen scope string (≤4 KiB)")
	supersedesFlag := fs.String("supersedes", "", "explicit ULID of a prior evolve event to supersede")
	kindDefFlag := fs.String("kind-definition", "", "meaning of kind; required on first use of a non-builtin kind")
	rationaleFlag := fs.String("rationale", "", "required explanation for this evolution event (≤64 KiB)")

	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(positional) < 1 {
		fmt.Fprintln(errOut, "mycelium evolve: <kind> is required")
		return ExitUsage
	}

	kind := positional[0]
	target := *targetFlag
	supersedesID := *supersedesFlag
	kindDef := *kindDefFlag
	rationale := *rationaleFlag

	// Validate kind.
	if rc := validateKind(kind, errOut); rc != ExitOK {
		return rc
	}

	// Validate rationale.
	if rationale == "" {
		fmt.Fprintln(errOut, "mycelium evolve: --rationale is required")
		return ExitReservedPrefix
	}
	if len(rationale) > maxRationaleSize {
		fmt.Fprintf(errOut, "mycelium evolve: --rationale exceeds %d bytes\n", maxRationaleSize)
		return ExitReservedPrefix
	}

	// Validate target.
	if len(target) > maxTargetSize {
		fmt.Fprintf(errOut, "mycelium evolve: --target exceeds %d bytes\n", maxTargetSize)
		return ExitReservedPrefix
	}

	id := ReadIdentity()
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium evolve: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	// Acquire mount-level lock for the entire evolve operation.
	release, err := acquireMountLock(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium evolve: acquire lock: %v\n", err)
		return ExitGenericError
	}
	defer release()

	// Load all prior evolve events from the activity log.
	priorEntries, err := loadEvolveEntries(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium evolve: scan activity log: %v\n", err)
		return ExitGenericError
	}

	// First-use enforcement for non-builtin kinds.
	if !isBuiltinKind(kind) && !kindHasBeenUsed(priorEntries, kind) {
		if kindDef == "" {
			fmt.Fprintf(errOut, "mycelium evolve: first use of kind %q requires --kind-definition\n", kind)
			return ExitReservedPrefix
		}
	}

	// Determine supersession.
	var resolvedSupersedes string

	if supersedesID != "" {
		// Explicit supersession: validate the referenced event exists and matches kind.
		prior, found := findByID(priorEntries, supersedesID)
		if !found {
			fmt.Fprintf(errOut, "mycelium evolve: --supersedes %q: no such evolve event\n", supersedesID)
			return ExitReservedPrefix
		}
		if prior.Kind != kind {
			fmt.Fprintf(errOut, "mycelium evolve: --supersedes %q: kind mismatch (event has kind %q, new event has kind %q)\n",
				supersedesID, prior.Kind, kind)
			return ExitReservedPrefix
		}
		resolvedSupersedes = supersedesID
	} else {
		// Implicit supersession by (kind, target) pair.
		if prior, found := latestActiveForKindTarget(priorEntries, kind, target); found {
			resolvedSupersedes = prior.ID
		}
	}

	// Mint a new ULID.
	newID := newULID()
	now := time.Now()

	// Write the primary evolve event.
	ev := EvolveEntry{
		ID:             newID,
		Kind:           kind,
		Target:         target,
		Supersedes:     resolvedSupersedes,
		KindDefinition: kindDef,
		Rationale:      rationale,
	}
	if rc := appendEvolveEntry(errOut, id, ev, now); rc != ExitOK {
		return rc
	}

	// If a kind_definition was supplied, write the synthetic _kind_definition chain event.
	if kindDef != "" {
		// Reload entries to include the one we just wrote (so supersession of
		// prior _kind_definition events is found correctly). We scan for
		// _kind_definition events separately: use the full set as collected so far
		// plus the one we wrote.
		allEntries := append(priorEntries, evolveLogEntry{
			ID:             newID,
			Kind:           kind,
			Target:         target,
			Supersedes:     resolvedSupersedes,
			KindDefinition: kindDef,
			Rationale:      rationale,
		})
		priorKDID := latestKindDefinitionID(allEntries, kind)

		kdID := newULID()
		kdEv := EvolveEntry{
			ID:         kdID,
			Kind:       reservedKindDefinition,
			Target:     kind,
			Supersedes: priorKDID,
			Rationale:  kindDef,
		}
		// Use a slightly later timestamp to ensure ordering.
		if rc := appendEvolveEntry(errOut, id, kdEv, now.Add(time.Nanosecond)); rc != ExitOK {
			return rc
		}
	}

	// Print result to stdout.
	if resolvedSupersedes != "" {
		fmt.Fprintf(out, `{"id":%q,"supersedes":%q}`+"\n", newID, resolvedSupersedes)
	} else {
		fmt.Fprintf(out, `{"id":%q}`+"\n", newID)
	}

	return ExitOK
}

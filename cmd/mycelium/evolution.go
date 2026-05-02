package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// evolutionKindRow is the JSON shape emitted by --kinds.
type evolutionKindRow struct {
	Name             string `json:"name"`
	Definition       string `json:"definition"`
	DefinedAtVersion string `json:"defined_at_version,omitempty"`
	Source           string `json:"source"`
	EventCount       int    `json:"event_count"`
}

// runEvolution is the handler for `mycelium evolution [flags]`.
func runEvolution(_ io.Reader, out, errOut io.Writer, args []string) int {
	fs := flag.NewFlagSet("evolution", flag.ContinueOnError)
	fs.SetOutput(errOut)

	kindFlag := fs.String("kind", "", "filter by kind name")
	sinceFlag := fs.String("since", "", "only events with ts >= DATE (RFC3339 or YYYY-MM-DD)")
	activeFlag := fs.Bool("active", false, "return only the latest non-superseded entry per (kind, target)")
	kindsFlag := fs.Bool("kinds", false, "enumerate distinct kinds")
	formatFlag := fs.String("format", "json", "output format: json|text")

	if _, err := parseInterspersed(fs, args); err != nil {
		return ExitUsage
	}

	// Validate mutual exclusions.
	if *activeFlag && *kindsFlag {
		fmt.Fprintln(errOut, "mycelium evolution: --active and --kinds are mutually exclusive")
		return ExitUsage
	}
	if *kindsFlag && *kindFlag != "" {
		fmt.Fprintln(errOut, "mycelium evolution: --kind cannot be used with --kinds")
		return ExitUsage
	}
	if *kindsFlag && *sinceFlag != "" {
		fmt.Fprintln(errOut, "mycelium evolution: --since cannot be used with --kinds")
		return ExitUsage
	}

	// Validate format.
	if *formatFlag != "json" && *formatFlag != "text" {
		fmt.Fprintln(errOut, "mycelium evolution: --format must be json or text")
		return ExitUsage
	}

	// Parse --since if provided.
	var sinceTime time.Time
	if *sinceFlag != "" {
		var err error
		sinceTime, err = parseSince(*sinceFlag)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium evolution: --since %q: %v\n", *sinceFlag, err)
			return ExitUsage
		}
	}

	id := ReadIdentity()
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium evolution: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	// Load all evolve entries from the activity log.
	// An empty or missing _activity directory is not an error — return empty results.
	entries, err := loadEvolveEntries(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium evolution: scan activity log: %v\n", err)
		return ExitGenericError
	}

	// Sort entries by id (ULID lexicographic = chronological).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	switch {
	case *kindsFlag:
		return runEvolutionKinds(out, errOut, entries, *formatFlag)
	case *activeFlag:
		return runEvolutionActive(out, errOut, entries, *kindFlag, sinceTime, *formatFlag)
	default:
		return runEvolutionDefault(out, errOut, entries, *kindFlag, sinceTime, *formatFlag)
	}
}

// parseSince parses a --since value, accepting RFC3339 timestamps and
// YYYY-MM-DD date-only strings. Any other format returns an error.
func parseSince(s string) (time.Time, error) {
	// Try RFC3339 first.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try RFC3339Nano.
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	// Try date-only: YYYY-MM-DD (treated as midnight UTC).
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("invalid format; accept RFC3339 (e.g. 2026-05-01T00:00:00Z) or date-only (e.g. 2026-05-01)")
}

// userFacingEntries filters out _kind_definition synthetic events from a slice.
func userFacingEntries(entries []evolveLogEntry) []evolveLogEntry {
	out := make([]evolveLogEntry, 0, len(entries))
	for _, e := range entries {
		if e.Kind != reservedKindDefinition {
			out = append(out, e)
		}
	}
	return out
}

// applyKindSinceFilters applies --kind and --since filters to a slice of entries.
func applyKindSinceFilters(entries []evolveLogEntry, kindFilter string, since time.Time) []evolveLogEntry {
	out := make([]evolveLogEntry, 0, len(entries))
	for _, e := range entries {
		if kindFilter != "" && e.Kind != kindFilter {
			continue
		}
		if !since.IsZero() {
			ts, err := time.Parse(time.RFC3339Nano, e.TS)
			if err != nil {
				// Try plain RFC3339.
				ts, err = time.Parse(time.RFC3339, e.TS)
				if err != nil {
					// Skip events with unparseable timestamps only if filtering by since.
					continue
				}
			}
			if ts.Before(since) {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

// runEvolutionDefault handles the default mode: stream all non-_kind_definition
// evolve events, optionally filtered by --kind and --since.
func runEvolutionDefault(out, errOut io.Writer, entries []evolveLogEntry, kindFilter string, since time.Time, format string) int {
	visible := userFacingEntries(entries)
	visible = applyKindSinceFilters(visible, kindFilter, since)

	for _, e := range visible {
		if err := printEvolveEntry(out, e, format); err != nil {
			fmt.Fprintf(errOut, "mycelium evolution: write output: %v\n", err)
			return ExitGenericError
		}
	}
	return ExitOK
}

// runEvolutionActive handles --active mode: per (kind, target) pair, return
// only the latest non-superseded entry.
func runEvolutionActive(out, errOut io.Writer, entries []evolveLogEntry, kindFilter string, since time.Time, format string) int {
	// Work on user-facing entries only (exclude _kind_definition).
	visible := userFacingEntries(entries)

	// Compute superseded IDs across the full user-facing set.
	sup := supersededIDs(visible)

	// Collect active entries: not in the superseded set.
	// We walk in order (already sorted chronologically) and keep a map of
	// (kind, target) -> latest active entry. Since entries are in chronological
	// order and we overwrite, the last one seen per pair wins.
	type kindTarget struct{ kind, target string }
	latest := make(map[kindTarget]evolveLogEntry)
	var order []kindTarget // to maintain stable output order

	for _, e := range visible {
		if sup[e.ID] {
			continue
		}
		kt := kindTarget{e.Kind, e.Target}
		if _, seen := latest[kt]; !seen {
			order = append(order, kt)
		}
		latest[kt] = e
	}

	// Apply filters and emit.
	for _, kt := range order {
		e := latest[kt]
		// Apply kind filter.
		if kindFilter != "" && e.Kind != kindFilter {
			continue
		}
		// Apply since filter.
		if !since.IsZero() {
			ts, err := time.Parse(time.RFC3339Nano, e.TS)
			if err != nil {
				ts, err = time.Parse(time.RFC3339, e.TS)
				if err != nil {
					continue
				}
			}
			if ts.Before(since) {
				continue
			}
		}
		if err := printEvolveEntry(out, e, format); err != nil {
			fmt.Fprintf(errOut, "mycelium evolution: write output: %v\n", err)
			return ExitGenericError
		}
	}
	return ExitOK
}

// runEvolutionKinds handles --kinds mode: enumerate distinct kinds.
func runEvolutionKinds(out, errOut io.Writer, entries []evolveLogEntry, format string) int {
	// Count user-facing events per kind (excludes _kind_definition synthetics).
	eventCounts := make(map[string]int)
	for _, e := range entries {
		if e.Kind == reservedKindDefinition {
			continue
		}
		eventCounts[e.Kind]++
	}

	// Build output rows.
	var rows []evolutionKindRow

	// Built-ins always first, in registry order.
	for _, b := range builtinKinds {
		rows = append(rows, evolutionKindRow{
			Name:             b.Name,
			Definition:       b.Definition,
			DefinedAtVersion: b.DefinedAtVersion,
			Source:           "builtin",
			EventCount:       eventCounts[b.Name],
		})
	}

	// Agent-introduced kinds: any kind in the log not in the builtin registry.
	// Use a set of builtin names for fast lookup.
	builtinSet := make(map[string]bool)
	for _, b := range builtinKinds {
		builtinSet[b.Name] = true
	}

	// Collect distinct agent kind names (preserving first-seen order for
	// alphabetical sort below).
	agentKindSeen := make(map[string]bool)
	var agentKindNames []string
	for _, e := range entries {
		if e.Kind == reservedKindDefinition {
			continue
		}
		if builtinSet[e.Kind] {
			continue
		}
		if !agentKindSeen[e.Kind] {
			agentKindSeen[e.Kind] = true
			agentKindNames = append(agentKindNames, e.Kind)
		}
	}
	sort.Strings(agentKindNames)

	// Build agent kind rows with the currently-active definition.
	for _, name := range agentKindNames {
		def := activeKindDefinition(entries, name)
		rows = append(rows, evolutionKindRow{
			Name:       name,
			Definition: def,
			Source:     "agent",
			EventCount: eventCounts[name],
		})
	}

	// Emit rows.
	if format == "json" {
		// Emit as JSON array.
		b, err := json.Marshal(rows)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium evolution: marshal kinds: %v\n", err)
			return ExitGenericError
		}
		fmt.Fprintf(out, "%s\n", b)
	} else {
		// Text: one row per line.
		for _, r := range rows {
			defLine := strings.SplitN(r.Definition, "\n", 2)[0]
			if len(defLine) > 60 {
				defLine = defLine[:57] + "..."
			}
			fmt.Fprintf(out, "%s\t%s\t%s\t%d\n", r.Name, r.Source, defLine, r.EventCount)
		}
	}

	return ExitOK
}

// activeKindDefinition returns the currently-active definition for an
// agent-introduced kind. It prefers the rationale of the latest active
// _kind_definition event targeting that kind; falls back to the kind_definition
// field on the first event of that kind.
func activeKindDefinition(entries []evolveLogEntry, kindName string) string {
	// Find the latest non-superseded _kind_definition event for this kind.
	sup := supersededIDs(entries)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Kind == reservedKindDefinition && e.Target == kindName && !sup[e.ID] {
			return e.Rationale
		}
	}
	// Fall back: first event of this kind with a kind_definition field.
	for _, e := range entries {
		if e.Kind == kindName && e.KindDefinition != "" {
			return e.KindDefinition
		}
	}
	return ""
}

// printEvolveEntry writes a single evolve entry to out in the requested format.
func printEvolveEntry(out io.Writer, e evolveLogEntry, format string) error {
	if format == "json" {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "%s\n", b)
		return err
	}
	// Text format: <ts>  <kind>  <target>  <id>  → <rationale first line>
	rationaleFirst := strings.SplitN(e.Rationale, "\n", 2)[0]
	_, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n",
		e.TS, e.Kind, e.Target, e.ID, rationaleFirst)
	return err
}

package mycelium

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// evolutionKindRow is the JSON shape emitted by evolve --kinds.
type evolutionKindRow struct {
	Name             string `json:"name"`
	Definition       string `json:"definition"`
	DefinedAtVersion string `json:"defined_at_version,omitempty"`
	Source           string `json:"source"`
	EventCount       int    `json:"event_count"`
}

type evolutionQueryOptions struct {
	Kind   string
	Since  string
	Active bool
	Kinds  bool
	List   bool
	Format string
}

// runEvolutionQuery handles the query modes owned by `mycelium evolve`:
// --list, --active, and --kinds.
func runEvolutionQuery(out, errOut io.Writer, opts evolutionQueryOptions) int {
	if opts.Active && opts.Kinds {
		fmt.Fprintln(errOut, "mycelium evolve: --active and --kinds are mutually exclusive")
		return ExitUsage
	}
	if opts.Active && opts.List {
		fmt.Fprintln(errOut, "mycelium evolve: --active and --list are mutually exclusive")
		return ExitUsage
	}
	if opts.Kinds && opts.List {
		fmt.Fprintln(errOut, "mycelium evolve: --kinds and --list are mutually exclusive")
		return ExitUsage
	}
	if opts.Kinds && opts.Kind != "" {
		fmt.Fprintln(errOut, "mycelium evolve: --kind cannot be used with --kinds")
		return ExitUsage
	}
	if opts.Kinds && opts.Since != "" {
		fmt.Fprintln(errOut, "mycelium evolve: --since cannot be used with --kinds")
		return ExitUsage
	}
	if !opts.Active && !opts.Kinds && !opts.List {
		fmt.Fprintln(errOut, "mycelium evolve: query mode requires --list, --active, or --kinds")
		return ExitUsage
	}
	if opts.Format == "" {
		opts.Format = "json"
	}
	if opts.Format != "json" && opts.Format != "text" {
		fmt.Fprintln(errOut, "mycelium evolve: --format must be json or text")
		return ExitUsage
	}

	var sinceTime time.Time
	if opts.Since != "" {
		var err error
		sinceTime, err = parseSince(opts.Since)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium evolve: --since %q: %v\n", opts.Since, err)
			return ExitUsage
		}
	}

	id := ReadIdentity()
	if id.Mount == "" {
		fmt.Fprintln(errOut, "mycelium evolve: MYCELIUM_MOUNT is not set")
		return ExitGenericError
	}

	entries, err := loadEvolveEntries(id.Mount)
	if err != nil {
		fmt.Fprintf(errOut, "mycelium evolve: scan activity log: %v\n", err)
		return ExitGenericError
	}

	// Sort entries by id (ULID lexicographic = chronological).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})

	switch {
	case opts.Kinds:
		return runEvolutionKinds(out, errOut, entries, opts.Format)
	case opts.Active:
		return runEvolutionActive(out, errOut, entries, opts.Kind, sinceTime, opts.Format)
	default:
		return runEvolutionDefault(out, errOut, entries, opts.Kind, sinceTime, opts.Format)
	}
}

// parseSince parses a --since value, accepting RFC3339 timestamps and
// YYYY-MM-DD date-only strings. Any other format returns an error.
func parseSince(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
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
				ts, err = time.Parse(time.RFC3339, e.TS)
				if err != nil {
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

// runEvolutionDefault handles --list mode: stream all non-_kind_definition
// evolve events, optionally filtered by --kind and --since.
func runEvolutionDefault(out, errOut io.Writer, entries []evolveLogEntry, kindFilter string, since time.Time, format string) int {
	visible := userFacingEntries(entries)
	visible = applyKindSinceFilters(visible, kindFilter, since)

	for _, e := range visible {
		if err := printEvolveEntry(out, e, format); err != nil {
			fmt.Fprintf(errOut, "mycelium evolve: write output: %v\n", err)
			return ExitGenericError
		}
	}
	return ExitOK
}

// runEvolutionActive handles --active mode. Targeted entries are active by
// supersession chain. Targetless entries are additive unless explicitly superseded.
func runEvolutionActive(out, errOut io.Writer, entries []evolveLogEntry, kindFilter string, since time.Time, format string) int {
	visible := userFacingEntries(entries)
	sup := supersededIDs(visible)

	type kindTarget struct{ kind, target string }
	latestTargeted := make(map[kindTarget]evolveLogEntry)
	var targetedOrder []kindTarget
	var active []evolveLogEntry

	for _, e := range visible {
		if sup[e.ID] {
			continue
		}
		if e.Target == "" {
			active = append(active, e)
			continue
		}
		kt := kindTarget{e.Kind, e.Target}
		if _, seen := latestTargeted[kt]; !seen {
			targetedOrder = append(targetedOrder, kt)
		}
		latestTargeted[kt] = e
	}

	for _, kt := range targetedOrder {
		active = append(active, latestTargeted[kt])
	}
	active = applyKindSinceFilters(active, kindFilter, since)

	for _, e := range active {
		if err := printEvolveEntry(out, e, format); err != nil {
			fmt.Fprintf(errOut, "mycelium evolve: write output: %v\n", err)
			return ExitGenericError
		}
	}
	return ExitOK
}

// runEvolutionKinds handles --kinds mode: enumerate distinct kinds.
func runEvolutionKinds(out, errOut io.Writer, entries []evolveLogEntry, format string) int {
	eventCounts := make(map[string]int)
	for _, e := range entries {
		if e.Kind == reservedKindDefinition {
			continue
		}
		eventCounts[e.Kind]++
	}

	var rows []evolutionKindRow
	for _, b := range builtinKinds {
		rows = append(rows, evolutionKindRow{
			Name:             b.Name,
			Definition:       b.Definition,
			DefinedAtVersion: b.DefinedAtVersion,
			Source:           "builtin",
			EventCount:       eventCounts[b.Name],
		})
	}

	builtinSet := make(map[string]bool)
	for _, b := range builtinKinds {
		builtinSet[b.Name] = true
	}

	agentKindSeen := make(map[string]bool)
	var agentKindNames []string
	for _, e := range entries {
		if e.Kind == reservedKindDefinition || builtinSet[e.Kind] {
			continue
		}
		if !agentKindSeen[e.Kind] {
			agentKindSeen[e.Kind] = true
			agentKindNames = append(agentKindNames, e.Kind)
		}
	}
	sort.Strings(agentKindNames)

	for _, name := range agentKindNames {
		rows = append(rows, evolutionKindRow{
			Name:       name,
			Definition: activeKindDefinition(entries, name),
			Source:     "agent",
			EventCount: eventCounts[name],
		})
	}

	if format == "json" {
		b, err := json.Marshal(rows)
		if err != nil {
			fmt.Fprintf(errOut, "mycelium evolve: marshal kinds: %v\n", err)
			return ExitGenericError
		}
		fmt.Fprintf(out, "%s\n", b)
		return ExitOK
	}

	for _, r := range rows {
		defLine := strings.SplitN(r.Definition, "\n", 2)[0]
		if len(defLine) > 60 {
			defLine = defLine[:57] + "..."
		}
		fmt.Fprintf(out, "%s\t%s\t%s\t%d\n", r.Name, r.Source, defLine, r.EventCount)
	}
	return ExitOK
}

// activeKindDefinition returns the currently-active definition for an
// agent-introduced kind. It prefers the rationale of the latest active
// _kind_definition event targeting kindName; falls back to the kind_definition
// field on the first event of that kind.
func activeKindDefinition(entries []evolveLogEntry, kindName string) string {
	sup := supersededIDs(entries)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Kind == reservedKindDefinition && e.Target == kindName && !sup[e.ID] {
			return e.Rationale
		}
	}
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
	rationaleFirst := strings.SplitN(e.Rationale, "\n", 2)[0]
	_, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n",
		e.TS, e.Kind, e.Target, e.ID, rationaleFirst)
	return err
}

package main

import (
	"encoding/json"
	"io"
	"math"
	"sort"
)

// Verdict is a detector's classification of a trajectory.
type Verdict struct {
	Detector string
	Healthy  bool
	Reason   string
}

// LoadTrajectory parses a JSONL stream into a slice of LogEntry.
func LoadTrajectory(r io.Reader) ([]LogEntry, error) {
	dec := json.NewDecoder(r)
	var entries []LogEntry
	for dec.More() {
		var e LogEntry
		if err := dec.Decode(&e); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// sessionGroups groups entries by session_id, ordered chronologically
// by each session's first timestamp.
func sessionGroups(entries []LogEntry) [][]LogEntry {
	type group struct {
		firstTS string
		items   []LogEntry
	}
	bySid := make(map[string]*group)
	var order []string
	for _, e := range entries {
		g, ok := bySid[e.SessionID]
		if !ok {
			g = &group{firstTS: e.TS}
			bySid[e.SessionID] = g
			order = append(order, e.SessionID)
		}
		g.items = append(g.items, e)
	}
	sort.SliceStable(order, func(i, j int) bool {
		return bySid[order[i]].firstTS < bySid[order[j]].firstTS
	})
	out := make([][]LogEntry, 0, len(order))
	for _, sid := range order {
		out = append(out, bySid[sid].items)
	}
	return out
}

// DetectWritesWithoutReads flags trajectories where the (write+edit)/read_signal
// ratio exceeds 0.7 across three or more consecutive sessions. A session with
// mutations and no read signals counts as +∞ ratio.
func DetectWritesWithoutReads(entries []LogEntry) Verdict {
	consecutive := 0
	for _, sess := range sessionGroups(entries) {
		var muts, reads int
		for _, e := range sess {
			switch e.Op {
			case "write", "edit":
				muts++
			case "read_signal":
				reads++
			}
		}
		var ratio float64
		switch {
		case muts == 0:
			ratio = 0
		case reads == 0:
			ratio = math.Inf(1)
		default:
			ratio = float64(muts) / float64(reads)
		}
		if ratio > 0.7 {
			consecutive++
			if consecutive >= 3 {
				return Verdict{
					Detector: "writes_without_reads",
					Healthy:  false,
					Reason:   "ratio >0.7 across ≥3 consecutive sessions",
				}
			}
		} else {
			consecutive = 0
		}
	}
	return Verdict{Detector: "writes_without_reads", Healthy: true}
}

// DetectNearDuplicatePaths flags trajectories where any single session
// contains three or more Levenshtein-1 path collisions across write entries.
func DetectNearDuplicatePaths(entries []LogEntry) Verdict {
	for _, sess := range sessionGroups(entries) {
		var paths []string
		for _, e := range sess {
			if e.Op == "write" && e.Path != "" {
				paths = append(paths, e.Path)
			}
		}
		collisions := 0
		for i := 0; i < len(paths); i++ {
			for j := i + 1; j < len(paths); j++ {
				if nearDuplicate(paths[i], paths[j]) {
					collisions++
				}
			}
		}
		if collisions >= 3 {
			return Verdict{
				Detector: "near_duplicate_paths",
				Healthy:  false,
				Reason:   "≥3 Levenshtein-1 path collisions in a single session",
			}
		}
	}
	return Verdict{Detector: "near_duplicate_paths", Healthy: true}
}

// DetectThrashing flags trajectories where any single session contains
// 50 or more activity-log entries.
func DetectThrashing(entries []LogEntry) Verdict {
	for _, sess := range sessionGroups(entries) {
		if len(sess) >= 50 {
			return Verdict{
				Detector: "thrashing",
				Healthy:  false,
				Reason:   "≥50 entries in a single session",
			}
		}
	}
	return Verdict{Detector: "thrashing", Healthy: true}
}

// RunDetectors evaluates all three failure-mode detectors against a trajectory.
func RunDetectors(entries []LogEntry) []Verdict {
	return []Verdict{
		DetectWritesWithoutReads(entries),
		DetectNearDuplicatePaths(entries),
		DetectThrashing(entries),
	}
}

// nearDuplicate reports whether a and b differ by exactly one byte-level
// insertion, deletion, or substitution.
func nearDuplicate(a, b string) bool {
	if a == b {
		return false
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	la, lb := len(a), len(b)
	if lb-la > 1 {
		return false
	}
	if la == lb {
		diffs := 0
		for i := 0; i < la; i++ {
			if a[i] != b[i] {
				diffs++
				if diffs > 1 {
					return false
				}
			}
		}
		return diffs == 1
	}
	for i := 0; i < la; i++ {
		if a[i] != b[i] {
			return a[i:] == b[i+1:]
		}
	}
	return true
}

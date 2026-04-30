package main

import (
	"os"
	"path/filepath"
	"testing"
)

// expectedVerdicts maps fixture filename to the expected (healthy bool) per
// detector. Detector ordering matches RunDetectors:
//
//	[0] writes_without_reads
//	[1] near_duplicate_paths
//	[2] thrashing
var expectedVerdicts = map[string][3]bool{
	"healthy.jsonl":                         {true, true, true},
	"unhealthy-writes-without-reads.jsonl":  {false, true, true},
	"unhealthy-duplicate-paths.jsonl":       {true, false, true},
	"unhealthy-thrashing.jsonl":             {true, true, false},
}

func loadFixture(t *testing.T, name string) []LogEntry {
	t.Helper()
	path := filepath.Join("testdata", "trajectories", name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %q: %v", name, err)
	}
	defer f.Close()
	entries, err := LoadTrajectory(f)
	if err != nil {
		t.Fatalf("parse fixture %q: %v", name, err)
	}
	return entries
}

func TestDetectors_ClassifyHandcraftedTrajectories(t *testing.T) {
	for name, want := range expectedVerdicts {
		t.Run(name, func(t *testing.T) {
			entries := loadFixture(t, name)
			got := RunDetectors(entries)
			if len(got) != 3 {
				t.Fatalf("RunDetectors returned %d verdicts, want 3", len(got))
			}
			for i, v := range got {
				if v.Healthy != want[i] {
					t.Errorf(
						"detector %q on %s: healthy=%v, want %v (reason: %q)",
						v.Detector, name, v.Healthy, want[i], v.Reason,
					)
				}
			}
		})
	}
}

func TestNearDuplicate(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"foo", "foo", false},
		{"foo", "fooo", true},
		{"foo", "fo", true},
		{"foo", "boo", true},
		{"foo", "bar", false},
		{"notes/glp1.md", "notes/glp-1.md", true},
		{"notes/glp1.md", "notes/glp1_.md", true},
		{"notes/glp1.md", "notes/glp10.md", true},
		{"notes/glp-1.md", "notes/glp1_.md", false},
		{"notes/glp1_.md", "notes/glp10.md", true},
		{"", "a", true},
		{"", "", false},
	}
	for _, c := range cases {
		got := nearDuplicate(c.a, c.b)
		if got != c.want {
			t.Errorf("nearDuplicate(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

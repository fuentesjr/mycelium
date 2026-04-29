package main

import (
	"testing"
)

func TestReadIdentityFromEnv(t *testing.T) {
	t.Setenv("MYCELIUM_AGENT_ID", "agent-123")
	t.Setenv("MYCELIUM_SESSION_ID", "session-abc")
	t.Setenv("MYCELIUM_MOUNT", "/tmp/mount")

	id := ReadIdentity()
	if id.AgentID != "agent-123" {
		t.Errorf("AgentID: got %q, want %q", id.AgentID, "agent-123")
	}
	if id.SessionID != "session-abc" {
		t.Errorf("SessionID: got %q, want %q", id.SessionID, "session-abc")
	}
	if id.Mount != "/tmp/mount" {
		t.Errorf("Mount: got %q, want %q", id.Mount, "/tmp/mount")
	}
}

func TestReadIdentityEmptyWhenUnset(t *testing.T) {
	t.Setenv("MYCELIUM_AGENT_ID", "")
	t.Setenv("MYCELIUM_SESSION_ID", "")
	t.Setenv("MYCELIUM_MOUNT", "")

	id := ReadIdentity()
	if id.AgentID != "" || id.SessionID != "" || id.Mount != "" {
		t.Errorf("expected empty identity when env unset, got %+v", id)
	}
}

package mycelium

import (
	"strings"
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

func TestReadIdentityDefaultsOptionalFieldsWhenUnset(t *testing.T) {
	t.Setenv("MYCELIUM_AGENT_ID", "")
	t.Setenv("MYCELIUM_SESSION_ID", "")
	t.Setenv("MYCELIUM_MOUNT", "")

	id := ReadIdentity()
	if id.AgentID != defaultAgentID {
		t.Errorf("AgentID default: got %q, want %q", id.AgentID, defaultAgentID)
	}
	if !strings.HasPrefix(id.SessionID, "auto-") {
		t.Errorf("SessionID default should be auto-generated, got %q", id.SessionID)
	}
	assertGeneratedID(t, id.SessionID, "auto")
	if id.Mount != "" {
		t.Errorf("Mount: got %q, want empty", id.Mount)
	}
}

func TestValidateAgentID(t *testing.T) {
	valid := []string{"", "agent-123", "pi_agent.1"}
	for _, agentID := range valid {
		if err := validateAgentID(agentID); err != nil {
			t.Errorf("validateAgentID(%q): unexpected error %v", agentID, err)
		}
	}

	invalid := []string{"../x", "a/b", `a\\b`, ".", "..", "bad:name", "bad\nname", "é"}
	for _, agentID := range invalid {
		if err := validateAgentID(agentID); err == nil {
			t.Errorf("validateAgentID(%q): got nil, want error", agentID)
		}
	}
}

func TestReadIdentityGeneratedSessionStableWithinProcess(t *testing.T) {
	t.Setenv("MYCELIUM_AGENT_ID", "")
	t.Setenv("MYCELIUM_SESSION_ID", "")
	t.Setenv("MYCELIUM_MOUNT", "/tmp/mount")

	first := ReadIdentity()
	second := ReadIdentity()
	if first.SessionID == "" || first.SessionID != second.SessionID {
		t.Fatalf("generated session should be stable within a process: first=%q second=%q", first.SessionID, second.SessionID)
	}
}

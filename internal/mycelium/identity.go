package mycelium

import (
	"os"
	"sync"
	"time"
)

const defaultAgentID = "agent"

var (
	defaultSessionOnce sync.Once
	defaultSessionID   string
)

type Identity struct {
	AgentID   string
	SessionID string
	Mount     string
}

func ReadIdentity() Identity {
	agentID := os.Getenv("MYCELIUM_AGENT_ID")
	if agentID == "" {
		agentID = defaultAgentID
	}

	sessionID := os.Getenv("MYCELIUM_SESSION_ID")
	if sessionID == "" {
		sessionID = generatedDefaultSessionID()
	}

	return Identity{
		AgentID:   agentID,
		SessionID: sessionID,
		Mount:     os.Getenv("MYCELIUM_MOUNT"),
	}
}

func generatedDefaultSessionID() string {
	defaultSessionOnce.Do(func() {
		defaultSessionID = newDefaultSessionID(time.Now())
	})
	return defaultSessionID
}

package mycelium

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

const defaultAgentID = "agent"

var ErrInvalidAgentID = errors.New("invalid agent id")

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

func validateAgentID(agentID string) error {
	if agentID == "" {
		return nil
	}
	if agentID == "." || agentID == ".." || len(agentID) > 128 {
		return ErrInvalidAgentID
	}
	for _, r := range agentID {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return fmt.Errorf("%w: must contain only ASCII letters, digits, '.', '-', or '_'", ErrInvalidAgentID)
		}
	}
	return nil
}

func generatedDefaultSessionID() string {
	defaultSessionOnce.Do(func() {
		defaultSessionID = newDefaultSessionID(time.Now())
	})
	return defaultSessionID
}

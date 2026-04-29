package main

import "os"

type Identity struct {
	AgentID   string
	SessionID string
	Mount     string
}

func ReadIdentity() Identity {
	return Identity{
		AgentID:   os.Getenv("MYCELIUM_AGENT_ID"),
		SessionID: os.Getenv("MYCELIUM_SESSION_ID"),
		Mount:     os.Getenv("MYCELIUM_MOUNT"),
	}
}

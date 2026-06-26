package mycelium

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var fallbackIDCounter atomic.Uint64

func newTxID(now time.Time) (string, error) {
	return newTimeRandomID("tx", now)
}

func newDefaultSessionID(now time.Time) string {
	id, err := newTimeRandomID("auto", now)
	if err == nil {
		return id
	}
	return fmt.Sprintf("auto-%019d-fallback-%016x", now.UTC().UnixNano(), fallbackIDCounter.Add(1))
}

func newTimeRandomID(prefix string, now time.Time) (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%019d-%s", prefix, now.UTC().UnixNano(), hex.EncodeToString(random[:])), nil
}

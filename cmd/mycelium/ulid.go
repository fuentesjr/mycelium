package main

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	ulidEntropy     = ulid.Monotonic(rand.Reader, 0)
	ulidEntropyOnce sync.Mutex
)

// newULID returns a new 26-character Crockford base32 ULID, monotonic within
// the process. Thread-safe.
func newULID() string {
	ulidEntropyOnce.Lock()
	id := ulid.MustNew(ulid.Timestamp(time.Now()), ulidEntropy)
	ulidEntropyOnce.Unlock()
	return id.String()
}

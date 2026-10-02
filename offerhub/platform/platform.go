// Package platform holds small building blocks owned by the Platform team.
// They stand in for real infrastructure (a database, feature flags, a clock).
package platform

import (
	"hash/fnv"
	"sync"
	"time"
)

// Now is the clock. Tests and the simulator replace it.
var Now = time.Now

// KV is a key-value store with versioned, conditional writes, like a database
// table that supports "write only if nobody changed it since I read it".
// Every call takes Latency, like a network round trip.
type KV struct {
	Latency time.Duration
	mu      sync.Mutex
	data    map[string]entry
}

type entry struct {
	val any
	ver int
}

func NewKV() *KV { return &KV{data: map[string]entry{}} }

// Get returns the value and its version. A missing key has version 0.
func (k *KV) Get(key string) (any, int, bool) {
	time.Sleep(k.Latency)
	k.mu.Lock()
	defer k.mu.Unlock()
	e, ok := k.data[key]
	return e.val, e.ver, ok
}

// Put writes val unconditionally (last write wins).
func (k *KV) Put(key string, val any) {
	time.Sleep(k.Latency)
	k.mu.Lock()
	defer k.mu.Unlock()
	k.data[key] = entry{val: val, ver: k.data[key].ver + 1}
}

// PutIf writes val only if the key's current version equals expected
// (0 = the key must not exist yet). It reports whether the write happened.
func (k *KV) PutIf(key string, val any, expected int) bool {
	time.Sleep(k.Latency)
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.data[key].ver != expected {
		return false
	}
	k.data[key] = entry{val: val, ver: expected + 1}
	return true
}

// Flags is a percentage-rollout feature flag store. Changing a percentage
// takes effect immediately, without a deploy.
type Flags struct {
	mu  sync.RWMutex
	pct map[string]int
}

func NewFlags() *Flags { return &Flags{pct: map[string]int{}} }

func (f *Flags) Set(name string, percent int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pct[name] = percent
}

// Enabled is sticky per user: the same user always gets the same answer
// for the same percentage.
func (f *Flags) Enabled(name, userID string) bool {
	f.mu.RLock()
	p := f.pct[name]
	f.mu.RUnlock()
	h := fnv.New32a()
	h.Write([]byte(name + ":" + userID))
	return int(h.Sum32()%100) < p
}

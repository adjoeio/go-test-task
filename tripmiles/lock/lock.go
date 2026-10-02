// Package lock stores the miles we locked in for a traveller when they booked a deal.
// Owned by Rewards Core (your team).
package lock

import "tripmiles/platform"

// Lock is written once, the first time a traveller books a deal.
// Miles per goal and the price are captured at that moment; later
// changes by the partner do not change an existing lock.
type Lock struct {
	UserID      string
	DealID      string
	PriceMicros int64
	Miles       map[string]int64 // goal name -> miles
}

type Store struct{ kv *platform.KV }

func NewStore(kv *platform.KV) *Store { return &Store{kv: kv} }

func key(userID, dealID string) string { return "lock|" + userID + "|" + dealID }

// Save stores l unless a lock for this traveller and deal already exists.
// It returns the lock that is in force (the existing one, if any).
func (s *Store) Save(l Lock) Lock {
	if s.kv.PutIf(key(l.UserID, l.DealID), l, 0) {
		return l
	}
	existing, _ := s.Get(l.UserID, l.DealID)
	return existing
}

func (s *Store) Get(userID, dealID string) (Lock, bool) {
	v, _, ok := s.kv.Get(key(userID, dealID))
	if !ok {
		return Lock{}, false
	}
	return v.(Lock), true
}

// Package payout pays travellers miles when they reach a trip goal.
// Owned by Rewards Core (your team).
//
// Updates come from partners through a queue that delivers at least once:
// the same update can arrive twice, at the same time, on different server
// instances. Every instance shares the same KV store; nothing else is shared.
package payout

import (
	"errors"

	"tripmiles/deals"
	"tripmiles/lock"
	"tripmiles/platform"
)

var (
	ErrNoLock      = errors.New("no booking lock for this traveller and deal")
	ErrUnknownGoal = errors.New("goal not in the lock")
)

// Update says a traveller reached a goal, e.g. "check-in".
type Update struct {
	ID     string // unique per update; a redelivery has the same ID
	UserID string
	DealID string
	Goal   string
}

// Service is one server instance. Several run at once against the same KV.
type Service struct {
	Locks       *lock.Store
	Deals       *deals.Store // deal settings as they are right now
	KV          *platform.KV
	FeePct      int64
	MilesPerUSD int64
}

func balanceKey(userID string) string { return "balance|" + userID }

// Pay credits the traveller for u and returns the miles paid by this call.
// Rules:
//   - pay exactly the miles locked when the deal was booked;
//   - no lock (or goal not in it) means no payment;
//   - each goal is paid at most once per traveller and deal,
//     however many times or wherever the update is delivered.
func (s *Service) Pay(u Update) (int64, error) {
	// TODO: implement. See the rules above and README.md.
	return 0, errors.New("not implemented")
}

// Balance returns the traveller's total miles.
func (s *Service) Balance(userID string) int64 {
	v, _, _ := s.KV.Get(balanceKey(userID))
	b, _ := v.(int64)
	return b
}

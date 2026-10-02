// Package budget keeps partner spend and stops deals that ran out of budget.
// Owned by Rewards Core (your team).
package budget

import (
	"errors"
	"sync"

	"tripmiles/deals"
)

var ErrUnavailable = errors.New("spend store unavailable")

// Spend is the shared spend counter (in production: a remote store that
// every server instance reads). It can be briefly unavailable.
type Spend struct {
	FailEvery int // every Nth call fails; 0 = never

	mu    sync.Mutex
	spent map[string]int64
	calls int
}

func NewSpend() *Spend { return &Spend{spent: map[string]int64{}} }

func (s *Spend) call() error {
	s.calls++
	if s.FailEvery > 0 && s.calls%s.FailEvery == 0 {
		return ErrUnavailable
	}
	return nil
}

// Get returns the spend of one deal. One network call.
func (s *Spend) Get(dealID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.call(); err != nil {
		return 0, err
	}
	return s.spent[dealID], nil
}

// GetMany returns the spend of many deals. One network call.
func (s *Spend) GetMany(dealIDs []string) (map[string]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.call(); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(dealIDs))
	for _, id := range dealIDs {
		out[id] = s.spent[id]
	}
	return out, nil
}

// Charge bills the partner for one booking.
func (s *Spend) Charge(dealID string, micros int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spent[dealID] += micros
}

// Calls is the number of spend lookups made so far.
func (s *Spend) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// Filter removes deals that cannot pay for one more booking
// (spent + price > budget).
func Filter(spend *Spend, ds []deals.Deal) []deals.Deal {
	// TODO: implement. See README.md.
	return ds
}

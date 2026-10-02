// Package budget keeps campaign spend and stops campaigns that ran out of budget.
// Owned by Offers Core (your team).
package budget

import (
	"errors"
	"sync"

	"offerhub/catalog"
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

// Get returns the spend of one campaign. One network call.
func (s *Spend) Get(campaignID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.call(); err != nil {
		return 0, err
	}
	return s.spent[campaignID], nil
}

// GetMany returns the spend of many campaigns. One network call.
func (s *Spend) GetMany(campaignIDs []string) (map[string]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.call(); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(campaignIDs))
	for _, id := range campaignIDs {
		out[id] = s.spent[id]
	}
	return out, nil
}

// Charge bills the advertiser for one install.
func (s *Spend) Charge(campaignID string, micros int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spent[campaignID] += micros
}

// Calls is the number of spend lookups made so far.
func (s *Spend) Calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// Filter removes campaigns that cannot pay for one more install
// (spent + CPI > budget).
func Filter(spend *Spend, cs []catalog.Campaign) []catalog.Campaign {
	// TODO: implement. See README.md.
	return cs
}

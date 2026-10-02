// Package catalog holds the advertisers' campaign configuration.
// Owned by the Campaign Management team. Advertisers can change a campaign at any time.
package catalog

import (
	"sort"
	"sync"
)

// Milestone is a step in the game the user is rewarded for, e.g. "level 10".
// Pct is its share of the campaign's total coins.
type Milestone struct {
	Name string
	Pct  int64
}

type Campaign struct {
	ID           string
	CPIMicros    int64 // what the advertiser pays per install, in micro-USD
	Countries    []string
	BudgetMicros int64 // total the advertiser is willing to spend
	Milestones   []Milestone
}

type Store struct {
	mu sync.RWMutex
	cs map[string]Campaign
}

func NewStore() *Store { return &Store{cs: map[string]Campaign{}} }

// Put creates or replaces a campaign (the advertiser saved new settings).
func (s *Store) Put(c Campaign) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cs[c.ID] = copyOf(c)
}

func (s *Store) Get(id string) (Campaign, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.cs[id]
	return copyOf(c), ok
}

// Live returns copies of all campaigns, safe to modify.
func (s *Store) Live() []Campaign {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Campaign, 0, len(s.cs))
	for _, c := range s.cs {
		out = append(out, copyOf(c))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func copyOf(c Campaign) Campaign {
	c.Countries = append([]string(nil), c.Countries...)
	c.Milestones = append([]Milestone(nil), c.Milestones...)
	return c
}

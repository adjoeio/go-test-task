// Package deals holds the travel partners' deal settings.
// Owned by the Partner Management team. Partners can change a deal at any time.
package deals

import (
	"sort"
	"sync"
)

// Goal is a step of the trip the traveller earns miles for, e.g. "check-in".
// Pct is its share of the deal's total miles.
type Goal struct {
	Name string
	Pct  int64
}

type Deal struct {
	ID                    string
	PricePerBookingMicros int64 // what the partner pays us per booking, in micro-USD
	Countries             []string
	BudgetMicros          int64 // total the partner is willing to spend
	Goals                 []Goal
}

type Store struct {
	mu sync.RWMutex
	ds map[string]Deal
}

func NewStore() *Store { return &Store{ds: map[string]Deal{}} }

// Put creates or replaces a deal (the partner saved new settings).
func (s *Store) Put(d Deal) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ds[d.ID] = copyOf(d)
}

func (s *Store) Get(id string) (Deal, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.ds[id]
	return copyOf(d), ok
}

// Live returns copies of all deals, safe to modify.
func (s *Store) Live() []Deal {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Deal, 0, len(s.ds))
	for _, d := range s.ds {
		out = append(out, copyOf(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func copyOf(d Deal) Deal {
	d.Countries = append([]string(nil), d.Countries...)
	d.Goals = append([]Goal(nil), d.Goals...)
	return d
}

// Package deals stores the settings of every deal and every partner.
// The Partner Management team owns this package. You can read it.
// You should not need to change it.
package deals

import "sort"

// A Deal is an offer from a partner, for example a cheap flight.
// Each booking of a deal gives the traveller one free hotel stay.
type Deal struct {
	ID        string
	Name      string
	PartnerID string

	// DailyLimit is the most hotel stays this deal gives away in one day.
	// TotalLimit is the most hotel stays this deal gives away in total.
	// A value of 0 means that there is no limit.
	DailyLimit int64
	TotalLimit int64
}

// A Partner can set a partner daily limit. This is the most hotel stays
// all its deals together give away in one day. A value of 0 means that there is no limit.
type Partner struct {
	ID         string
	Name       string
	DailyLimit int64
}

// Store holds the deals and the partners. It does not change while the app runs.
type Store struct {
	deals    []Deal
	partners map[string]Partner
}

func NewStore(ds []Deal, ps []Partner) *Store {
	s := &Store{deals: ds, partners: map[string]Partner{}}
	for _, p := range ps {
		s.partners[p.ID] = p
	}
	return s
}

// All returns all deals.
func (s *Store) All() []Deal {
	out := append([]Deal(nil), s.deals...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Partner returns the settings of one partner.
func (s *Store) Partner(id string) Partner {
	return s.partners[id]
}

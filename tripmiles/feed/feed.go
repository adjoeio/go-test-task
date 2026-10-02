// Package feed builds the deal list a traveller sees. Owned by Rewards Core (your team).
//
// Pipeline: country filter -> budget filter -> rank -> miles -> lock.
package feed

import (
	"sort"

	"tripmiles/budget"
	"tripmiles/deals"
	"tripmiles/lock"
	"tripmiles/miles"
)

type Item struct {
	DealID string
	Miles  map[string]int64 // goal -> miles the traveller will get
}

type Service struct {
	Deals       *deals.Store
	Locks       *lock.Store
	Spend       *budget.Spend
	FeePct      int64
	MilesPerUSD int64
}

// List returns the deals for a traveller. In this small app, being shown a
// deal counts as booking it, so the miles are locked here.
func (s *Service) List(userID, country string) []Item {
	ds := s.Deals.Live()
	ds = byCountry(ds, country)
	ds = budget.Filter(s.Spend, ds)
	ds = rank(ds)

	out := make([]Item, 0, len(ds))
	for _, d := range ds {
		total := miles.Miles(d.PricePerBookingMicros, s.FeePct, s.MilesPerUSD)
		l := lock.Lock{UserID: userID, DealID: d.ID, PriceMicros: d.PricePerBookingMicros, Miles: map[string]int64{}}
		for _, g := range d.Goals {
			l.Miles[g.Name] = total * g.Pct / 100
		}
		// If the traveller booked this deal before, the old lock stands.
		l = s.Locks.Save(l)
		out = append(out, Item{DealID: d.ID, Miles: l.Miles})
	}
	return out
}

func byCountry(ds []deals.Deal, country string) []deals.Deal {
	out := ds[:0]
	for _, d := range ds {
		for _, c := range d.Countries {
			if c == country {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

// rank puts the best-paying deals first.
func rank(ds []deals.Deal) []deals.Deal {
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].PricePerBookingMicros > ds[j].PricePerBookingMicros })
	return ds
}

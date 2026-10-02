// Package deals keeps the settings of every deal and partner.
// Owned by the Partner Management team. You can read it, but you should not need to change it.
package deals

import "sort"

// A Deal is an offer from a travel company (the partner), for example a cheap flight.
// Money is in cents: 400 means $4.00.
type Deal struct {
	ID        string
	Name      string
	PartnerID string
	Countries []string // the countries where travellers can see this deal

	// PriceCents is what the partner pays us for each booking.
	// Some countries have a different price, for example Germany.
	PriceCents        int64
	CountryPriceCents map[string]int64

	// The most the partner will pay us for this deal in one day, and in total.
	// 0 means there is no limit.
	DailyBudgetCents int64
	TotalBudgetCents int64
}

// PriceIn returns what the partner pays us for one booking from a traveller in this country.
func (d Deal) PriceIn(country string) int64 {
	if p, ok := d.CountryPriceCents[country]; ok {
		return p
	}
	return d.PriceCents
}

// A Partner can set one daily limit for all its deals together. 0 means no limit.
type Partner struct {
	ID              string
	Name            string
	DailyLimitCents int64
}

// Store holds the deals and partners. It does not change while the app runs.
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

// AvailableIn returns the deals that travellers in this country can see.
func (s *Store) AvailableIn(country string) []Deal {
	var out []Deal
	for _, d := range s.deals {
		for _, c := range d.Countries {
			if c == country {
				out = append(out, d)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) Partner(id string) Partner {
	return s.partners[id]
}

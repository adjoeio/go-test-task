// Package deals stores the settings of every deal and every partner.
// The Partner Management team owns this package. You can read it.
// You should not need to change it.
package deals

import "sort"

// A Deal is an offer from a partner, for example a cheap flight.
// A partner is a travel company. All money is in cents: 400 means $4.00.
type Deal struct {
	ID        string
	Name      string
	PartnerID string
	Countries []string // Travellers in these countries can see the deal.

	// PriceCents is the amount the partner pays us for one booking.
	// CountryPriceCents replaces that amount in some countries.
	PriceCents        int64
	CountryPriceCents map[string]int64

	// DailyBudgetCents is the most the partner pays for this deal in one day.
	// TotalBudgetCents is the most the partner pays for this deal in total.
	// A value of 0 means that there is no budget.
	DailyBudgetCents int64
	TotalBudgetCents int64
}

// PriceIn returns the amount the partner pays us for one booking
// from a traveller in this country.
func (d Deal) PriceIn(country string) int64 {
	if p, ok := d.CountryPriceCents[country]; ok {
		return p
	}
	return d.PriceCents
}

// A Partner can set a partner daily limit. This is the most the partner pays
// for all its deals together in one day. A value of 0 means that there is no limit.
type Partner struct {
	ID              string
	Name            string
	DailyLimitCents int64
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

// Partner returns the settings of one partner.
func (s *Store) Partner(id string) Partner {
	return s.partners[id]
}

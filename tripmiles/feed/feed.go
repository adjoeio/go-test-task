// Package feed builds the list of deals that a traveller sees.
// The Offers team owns this package. The Offers team is your team.
package feed

import (
	"sort"

	"tripmiles/deals"
	"tripmiles/stats"
)

// Feed runs on one app server. Each server has its own Feed and its own stats Client.
type Feed struct {
	Deals *deals.Store
	Stats *stats.Client
}

// List runs each time a traveller opens the app.
// It returns the deals that the traveller can see. The deal that pays most is first.
func (f *Feed) List(country string) []deals.Deal {
	ds := f.Deals.AvailableIn(country)
	ds = f.dealBudgetCheck(ds)
	ds = f.partnerLimitCheck(ds)
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].PriceIn(country) > ds[j].PriceIn(country) })
	return ds
}

// Book records that a traveller booked a deal.
func (f *Feed) Book(d deals.Deal, country string) {
	f.Stats.RecordBooking(d.ID, d.PartnerID, d.PriceIn(country))
}

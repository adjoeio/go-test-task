// Package offers builds the offer list a user sees. Owned by Offers Core (your team).
//
// Pipeline: country filter -> budget filter -> rank -> coins -> promise.
package offers

import (
	"sort"

	"offerhub/budget"
	"offerhub/catalog"
	"offerhub/coins"
	"offerhub/promise"
)

type Offer struct {
	CampaignID string
	Coins      map[string]int64 // milestone -> coins the user will get
}

type Service struct {
	Catalog     *catalog.Store
	Promises    *promise.Store
	Spend       *budget.Spend
	MarginPct   int64
	CoinsPerUSD int64
}

func (s *Service) List(userID, country string) []Offer {
	cs := s.Catalog.Live()
	cs = byCountry(cs, country)
	cs = budget.Filter(s.Spend, cs)
	cs = rank(cs)

	out := make([]Offer, 0, len(cs))
	for _, c := range cs {
		total := coins.Coins(c.CPIMicros, s.MarginPct, s.CoinsPerUSD)
		p := promise.Promise{UserID: userID, CampaignID: c.ID, CPIMicros: c.CPIMicros, Coins: map[string]int64{}}
		for _, m := range c.Milestones {
			p.Coins[m.Name] = total * m.Pct / 100
		}
		// If the user was shown this campaign before, the old promise stands.
		p = s.Promises.Commit(p)
		out = append(out, Offer{CampaignID: c.ID, Coins: p.Coins})
	}
	return out
}

func byCountry(cs []catalog.Campaign, country string) []catalog.Campaign {
	out := cs[:0]
	for _, c := range cs {
		for _, cc := range c.Countries {
			if cc == country {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// rank puts the best-paying campaigns first.
func rank(cs []catalog.Campaign) []catalog.Campaign {
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].CPIMicros > cs[j].CPIMicros })
	return cs
}

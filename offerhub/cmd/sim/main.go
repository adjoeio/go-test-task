// Command sim replays a busy hour of OfferHub traffic and checks the business rules.
//
//	go run ./cmd/sim
//
// It prints one CHECK line per rule and exits non-zero if any rule is broken.
package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"offerhub/budget"
	"offerhub/catalog"
	"offerhub/offers"
	"offerhub/platform"
	"offerhub/promise"
	"offerhub/reward"
)

const (
	users        = 200
	instances    = 4    // reward server instances running at the same time
	duplicateEvt = 0.25 // share of events the queue delivers twice
)

func main() {
	rng := rand.New(rand.NewSource(42))
	kv := platform.NewKV()
	kv.Latency = 200 * time.Microsecond
	cat := catalog.NewStore()
	promises := promise.NewStore(kv)
	spend := budget.NewSpend()
	spend.FailEvery = 5 // the spend store is flaky

	for i, cpi := range []int64{3_000_000, 2_500_000, 2_000_000, 1_500_000, 1_000_000, 500_000} {
		cat.Put(catalog.Campaign{
			ID:           fmt.Sprintf("game-%d", i+1),
			CPIMicros:    cpi,
			Countries:    []string{"DE", "US"},
			BudgetMicros: 20 * cpi, // room for 20 installs
			Milestones:   []catalog.Milestone{{Name: "level-5", Pct: 30}, {Name: "level-10", Pct: 70}},
		})
	}
	svc := &offers.Service{Catalog: cat, Promises: promises, Spend: spend, MarginPct: 30, CoinsPerUSD: 1000}

	// 1. Users open the offerwall; each installs the top offer, if any.
	type install struct{ user, campaign string }
	var installs []install
	for u := 0; u < users; u++ {
		user := fmt.Sprintf("user-%03d", u)
		list := svc.List(user, []string{"DE", "US"}[u%2])
		if len(list) == 0 {
			continue
		}
		c := list[0].CampaignID
		p, _ := promises.Get(user, c)
		spend.Charge(c, p.CPIMicros) // the advertiser is billed per install
		installs = append(installs, install{user, c})
	}
	lookups := spend.Calls()
	budgets := map[string]int64{}
	for _, c := range cat.Live() {
		budgets[c.ID] = c.BudgetMicros
	}

	// 2. Advertisers change their campaigns while users are still playing.
	for _, c := range cat.Live() {
		c.CPIMicros *= 2
		c.Milestones = []catalog.Milestone{{Name: "level-5", Pct: 70}, {Name: "level-10", Pct: 30}}
		cat.Put(c)
	}

	// 3. Users reach milestones. Fast players reach both milestones within
	// moments, the queue sometimes redelivers an event right away, and some
	// events arrive for users we never showed an offer to.
	var events []reward.Event
	for i, in := range installs {
		for _, m := range []string{"level-5", "level-10"} {
			e := reward.Event{ID: in.user + "|" + in.campaign + "|" + m, UserID: in.user, CampaignID: in.campaign, Milestone: m}
			events = append(events, e)
			if rng.Float64() < duplicateEvt {
				events = append(events, e)
			}
		}
		if i%12 == 0 {
			user := fmt.Sprintf("stranger-%d", i/12)
			events = append(events, reward.Event{ID: user, UserID: user, CampaignID: "game-1", Milestone: "level-5"})
		}
	}

	queue := make(chan reward.Event)
	var wg sync.WaitGroup
	for i := 0; i < instances; i++ {
		inst := &reward.Service{Promises: promises, Catalog: cat, KV: kv, MarginPct: 30, CoinsPerUSD: 1000}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range queue {
				inst.Pay(e)
			}
		}()
	}
	for _, e := range events {
		queue <- e
	}
	close(queue)
	wg.Wait()

	// 4. Check the business rules.
	r := &reward.Service{Promises: promises, Catalog: cat, KV: kv}
	wrongPay := 0
	for _, in := range installs {
		p, _ := promises.Get(in.user, in.campaign)
		if r.Balance(in.user) != p.Coins["level-5"]+p.Coins["level-10"] {
			wrongPay++
		}
	}
	strangersPaid := 0
	for g := 0; g*12 < len(installs); g++ {
		if r.Balance(fmt.Sprintf("stranger-%d", g)) != 0 {
			strangersPaid++
		}
	}
	overspent := 0
	spend.FailEvery = 0
	for id, b := range budgets {
		spent, _ := spend.Get(id)
		if spent > b {
			overspent++
		}
	}

	fmt.Printf("offer requests: %d, installs: %d, reward events: %d\n\n", users, len(installs), len(events))
	ok := check("every user is paid exactly what they were promised", wrongPay == 0, fmt.Sprintf("%d of %d users paid a different amount", wrongPay, len(installs)))
	ok = check("users without a promise are paid nothing", strangersPaid == 0, fmt.Sprintf("%d strangers paid", strangersPaid)) && ok
	ok = check("no campaign spends more than its budget", overspent == 0, fmt.Sprintf("%d campaigns over budget", overspent)) && ok
	ok = check("at most one spend lookup per offer request", lookups <= users, fmt.Sprintf("%d lookups for %d requests", lookups, users)) && ok
	if !ok {
		os.Exit(1)
	}
}

func check(rule string, pass bool, detail string) bool {
	if pass {
		fmt.Printf("CHECK %-55s PASS\n", rule)
	} else {
		fmt.Printf("CHECK %-55s FAIL  (%s)\n", rule, detail)
	}
	return pass
}

// Command sim replays a busy hour of TripMiles traffic and checks the business rules.
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

	"tripmiles/budget"
	"tripmiles/deals"
	"tripmiles/feed"
	"tripmiles/lock"
	"tripmiles/payout"
	"tripmiles/platform"
)

const (
	travellers   = 200
	instances    = 4    // payout server instances running at the same time
	duplicateUpd = 0.25 // share of partner updates delivered twice
)

func main() {
	rng := rand.New(rand.NewSource(42))
	kv := platform.NewKV()
	kv.Latency = 200 * time.Microsecond
	ds := deals.NewStore()
	locks := lock.NewStore(kv)
	spend := budget.NewSpend()
	spend.FailEvery = 5 // the spend store is flaky

	partners := []string{"skyline-air", "harbor-hotels", "nordic-rail", "sunset-resorts", "metro-cabs", "alpine-lodge"}
	for i, price := range []int64{3_000_000, 2_500_000, 2_000_000, 1_500_000, 1_000_000, 500_000} {
		ds.Put(deals.Deal{
			ID:                    partners[i],
			PricePerBookingMicros: price,
			Countries:             []string{"DE", "US"},
			BudgetMicros:          20 * price, // room for 20 bookings
			Goals:                 []deals.Goal{{Name: "check-in", Pct: 30}, {Name: "trip-completed", Pct: 70}},
		})
	}
	svc := &feed.Service{Deals: ds, Locks: locks, Spend: spend, FeePct: 30, MilesPerUSD: 1000}

	// 1. Travellers open the app; each books the top deal, if any.
	type booking struct{ user, deal string }
	var bookings []booking
	for t := 0; t < travellers; t++ {
		user := fmt.Sprintf("traveller-%03d", t)
		list := svc.List(user, []string{"DE", "US"}[t%2])
		if len(list) == 0 {
			continue
		}
		d := list[0].DealID
		l, _ := locks.Get(user, d)
		spend.Charge(d, l.PriceMicros) // the partner is billed per booking
		bookings = append(bookings, booking{user, d})
	}
	lookups := spend.Calls()
	budgets := map[string]int64{}
	for _, d := range ds.Live() {
		budgets[d.ID] = d.BudgetMicros
	}

	// 2. Partners change their deals while travellers are still on their trips.
	for _, d := range ds.Live() {
		d.PricePerBookingMicros *= 2
		d.Goals = []deals.Goal{{Name: "check-in", Pct: 70}, {Name: "trip-completed", Pct: 30}}
		ds.Put(d)
	}

	// 3. Partners send trip updates. Short trips finish within moments, the
	// queue sometimes redelivers an update right away, and some updates
	// arrive for travellers who never booked.
	var updates []payout.Update
	for i, b := range bookings {
		for _, g := range []string{"check-in", "trip-completed"} {
			u := payout.Update{ID: b.user + "|" + b.deal + "|" + g, UserID: b.user, DealID: b.deal, Goal: g}
			updates = append(updates, u)
			if rng.Float64() < duplicateUpd {
				updates = append(updates, u)
			}
		}
		if i%12 == 0 {
			user := fmt.Sprintf("stranger-%d", i/12)
			updates = append(updates, payout.Update{ID: user, UserID: user, DealID: partners[0], Goal: "check-in"})
		}
	}

	queue := make(chan payout.Update)
	var wg sync.WaitGroup
	for i := 0; i < instances; i++ {
		inst := &payout.Service{Locks: locks, Deals: ds, KV: kv, FeePct: 30, MilesPerUSD: 1000}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range queue {
				inst.Pay(u)
			}
		}()
	}
	for _, u := range updates {
		queue <- u
	}
	close(queue)
	wg.Wait()

	// 4. Check the business rules.
	p := &payout.Service{Locks: locks, Deals: ds, KV: kv}
	wrongPay := 0
	for _, b := range bookings {
		l, _ := locks.Get(b.user, b.deal)
		if p.Balance(b.user) != l.Miles["check-in"]+l.Miles["trip-completed"] {
			wrongPay++
		}
	}
	strangersPaid := 0
	for g := 0; g*12 < len(bookings); g++ {
		if p.Balance(fmt.Sprintf("stranger-%d", g)) != 0 {
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

	fmt.Printf("app opens: %d, bookings: %d, partner updates: %d\n\n", travellers, len(bookings), len(updates))
	ok := check("every traveller is paid exactly the miles locked at booking", wrongPay == 0, fmt.Sprintf("%d of %d travellers paid a different amount", wrongPay, len(bookings)))
	ok = check("travellers without a booking are paid nothing", strangersPaid == 0, fmt.Sprintf("%d strangers paid", strangersPaid)) && ok
	ok = check("no deal spends more than its partner budget", overspent == 0, fmt.Sprintf("%d deals over budget", overspent)) && ok
	ok = check("at most one spend lookup per app open", lookups <= travellers, fmt.Sprintf("%d lookups for %d app opens", lookups, travellers)) && ok
	if !ok {
		os.Exit(1)
	}
}

func check(rule string, pass bool, detail string) bool {
	if pass {
		fmt.Printf("CHECK %-62s PASS\n", rule)
	} else {
		fmt.Printf("CHECK %-62s FAIL  (%s)\n", rule, detail)
	}
	return pass
}

// Command sim plays one busy day of TripMiles and checks the four rules from README.md.
//
//	go run ./cmd/sim
package main

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"tripmiles/deals"
	"tripmiles/feed"
	"tripmiles/stats"
)

const (
	appOpens  = 3000 // travellers opening the app during the day
	servers   = 4    // app servers, all sharing one counter service
	waiting   = 150  // partners confirm all bookings except the newest 150
	failEvery = 100  // every 100th question to the counter service gets no answer
	msPerAsk  = 1    // each question to the counter service takes about 1 ms
)

func main() {
	store := dayDeals()
	svc := stats.NewService()
	svc.FailEveryNth = failEvery

	r := playTheDay(store, svc)
	ok := checkTheRules(store, svc, r)
	if !ok {
		os.Exit(1)
	}
}

// dayDeals are the deals for the simulated day. They are the same 6 deals as in
// README.md, but with bigger budgets because the day is busier.
func dayDeals() *deals.Store {
	everywhere := []string{"DE", "US"}
	return deals.NewStore(
		[]deals.Deal{
			{ID: "skyline-lisbon", Name: "Skyline Air: Berlin to Lisbon", PartnerID: "skyline", Countries: everywhere,
				PriceCents: 400, CountryPriceCents: map[string]int64{"DE": 500}, DailyBudgetCents: 200000},
			{ID: "skyline-rome", Name: "Skyline Air: Berlin to Rome", PartnerID: "skyline", Countries: everywhere,
				PriceCents: 300, DailyBudgetCents: 120000},
			{ID: "harbor-paris", Name: "Harbor Hotels: Paris", PartnerID: "harbor", Countries: everywhere,
				PriceCents: 600, DailyBudgetCents: 150000},
			{ID: "harbor-vienna", Name: "Harbor Hotels: Vienna", PartnerID: "harbor", Countries: everywhere,
				PriceCents: 350, DailyBudgetCents: 140000},
			{ID: "nordic-oslo", Name: "Nordic Rail: Oslo to Bergen", PartnerID: "nordic", Countries: everywhere,
				PriceCents: 400, DailyBudgetCents: 160000},
			{ID: "nordic-stockholm", Name: "Nordic Rail: Stockholm to Oslo", PartnerID: "nordic", Countries: everywhere,
				PriceCents: 250, DailyBudgetCents: 100000},
		},
		[]deals.Partner{
			{ID: "skyline", Name: "Skyline Air"},
			{ID: "harbor", Name: "Harbor Hotels"},
			{ID: "nordic", Name: "Nordic Rail", DailyLimitCents: 200000},
		},
	)
}

type results struct {
	asksPerOpen         []int // how many questions each app open asked
	affordableButHidden int   // app opens that left out a deal that was clearly affordable
	firstBudgetUsedUpAt int   // minute of the day, -1 if never
}

// playTheDay sends the app opens to the 4 servers. Each traveller books the
// first deal on their list, if there is one.
func playTheDay(store *deals.Store, svc *stats.Service) results {
	var (
		mu sync.Mutex
		r  = results{firstBudgetUsedUpAt: -1}
		wg sync.WaitGroup
	)
	for s := 0; s < servers; s++ {
		wg.Add(1)
		go func(server int) {
			defer wg.Done()
			f := &feed.Feed{Deals: store, Stats: svc.NewClient()}
			for open := server; open < appOpens; open += servers {
				country := []string{"DE", "US"}[open%2]
				minute := open * 24 * 60 / appOpens

				before := f.Stats.Calls()
				list := f.List(country)
				asked := f.Stats.Calls() - before
				hidden := affordableButHidden(store, svc, country, list)

				usedUp := false
				if len(list) > 0 && open%2 == 0 { // every second traveller books
					f.Book(list[0], country)
					usedUp = budgetUsedUp(svc, list[0], country)
				}
				svc.ConfirmAllButNewest(waiting)

				mu.Lock()
				r.asksPerOpen = append(r.asksPerOpen, asked)
				if hidden {
					r.affordableButHidden++
				}
				if usedUp && (r.firstBudgetUsedUpAt == -1 || minute < r.firstBudgetUsedUpAt) {
					r.firstBudgetUsedUpAt = minute
				}
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()
	return r
}

// affordableButHidden says whether the list left out a deal that had used
// less than 80% of its budget and whose partner had used less than 80% of its limit.
func affordableButHidden(store *deals.Store, svc *stats.Service, country string, list []deals.Deal) bool {
	shown := map[string]bool{}
	for _, d := range list {
		shown[d.ID] = true
	}
	for _, d := range store.AvailableIn(country) {
		dealSpent := svc.TrueNumbers(d.ID).BookedTodayCents
		limit := store.Partner(d.PartnerID).DailyLimitCents
		partnerSpent := svc.TruePartnerBookedTodayCents(d.PartnerID)
		clearlyAffordable := dealSpent*10 < d.DailyBudgetCents*8 && (limit == 0 || partnerSpent*10 < limit*8)
		if clearlyAffordable && !shown[d.ID] {
			return true
		}
	}
	return false
}

func budgetUsedUp(svc *stats.Service, d deals.Deal, country string) bool {
	return svc.TrueNumbers(d.ID).BookedTodayCents+d.PriceIn(country) > d.DailyBudgetCents
}

func checkTheRules(store *deals.Store, svc *stats.Service, r results) bool {
	total := 0
	for _, a := range r.asksPerOpen {
		total += a
	}
	sort.Ints(r.asksPerOpen)
	p99ms := r.asksPerOpen[len(r.asksPerOpen)*99/100] * msPerAsk
	avgAsks := float64(total) / float64(len(r.asksPerOpen))

	worstName, worstSpent, worstBudget := "", int64(0), int64(1)
	consider := func(name string, spent, budget int64) {
		if budget > 0 && spent*worstBudget > worstSpent*budget {
			worstName, worstSpent, worstBudget = name, spent, budget
		}
	}
	partners := map[string]bool{}
	for _, d := range store.AvailableIn("DE") {
		consider(d.Name, svc.TrueNumbers(d.ID).BookedTodayCents, d.DailyBudgetCents)
		partners[d.PartnerID] = true
	}
	for id := range partners {
		p := store.Partner(id)
		consider(p.Name+" (all deals together)", svc.TruePartnerBookedTodayCents(id), p.DailyLimitCents)
	}
	overPct := float64(worstSpent-worstBudget) * 100 / float64(worstBudget)

	fmt.Printf("A simulated day: %d app opens on %d servers. Partners confirm bookings late,\n", appOpens, servers)
	fmt.Printf("and 1 in %d questions to the shared counter service gets no answer.\n\n", failEvery)

	ok := check("Each app open asks the shared counter service at most once",
		total <= len(r.asksPerOpen),
		fmt.Sprintf("On average it asked %.1f times.", avgAsks))
	ok = check("99 out of 100 app opens take 15 ms or less",
		p99ms <= 15,
		fmt.Sprintf("The slowest 1 in 100 took %d ms.", p99ms)) && ok
	ok = check("No deal or partner goes more than 5% over its daily budget",
		overPct <= 5,
		fmt.Sprintf("The worst was %s: spent $%.2f of $%.2f (%.1f%% over).",
			worstName, float64(worstSpent)/100, float64(worstBudget)/100, overPct)) && ok
	ok = check("Travellers always see the deals that can still be afforded",
		r.affordableButHidden == 0,
		fmt.Sprintf("%d app opens left out a deal that had used less than 80%% of its budget.", r.affordableButHidden)) && ok

	if r.firstBudgetUsedUpAt >= 0 {
		fmt.Printf("\nINFO   The first daily budget was used up at %02d:%02d.\n", r.firstBudgetUsedUpAt/60, r.firstBudgetUsedUpAt%60)
	}
	return ok
}

func check(rule string, pass bool, detail string) bool {
	if pass {
		fmt.Printf("CHECK  %-62s PASS\n", rule)
	} else {
		fmt.Printf("CHECK  %-62s FAIL  %s\n", rule, detail)
	}
	return pass
}

// Command sim runs one busy day of TripMiles. Then it checks the four rules in README.md.
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
	appOpens     = 3000 // number of app opens in the day
	servers      = 4    // number of app servers; all use one shared counter service
	unconfirmed  = 150  // partners confirm all bookings except the newest 150
	failEvery    = 100  // the service does not answer every 100th request
	msPerRequest = 1    // one request to the service takes about 1 ms
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

// dayDeals returns the deals for the simulated day. They are the 6 deals from
// README.md. The budgets are larger because the day has more traffic.
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
	requestsPerOpen     []int // the number of requests that each app open sent
	affordableButHidden int   // the number of app opens that removed an affordable deal
	firstBudgetUsedUpAt int   // the minute of the day; -1 if no budget was used
}

// playTheDay sends the app opens to the 4 servers.
// Every second traveller books the first deal on the list, if the list is not empty.
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
				sent := f.Stats.Calls() - before
				hidden := affordableButHidden(store, svc, country, list)

				usedUp := false
				if len(list) > 0 && open%2 == 0 { // every second traveller books
					f.Book(list[0], country)
					usedUp = budgetUsedUp(svc, list[0], country)
				}
				svc.ConfirmAllButNewest(unconfirmed)

				mu.Lock()
				r.requestsPerOpen = append(r.requestsPerOpen, sent)
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

// affordableButHidden returns true if the list does not have an affordable deal.
// A deal is affordable if it has used less than 80% of its daily budget, and its
// partner has used less than 80% of its partner daily limit.
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
	for _, n := range r.requestsPerOpen {
		total += n
	}
	sort.Ints(r.requestsPerOpen)
	p99ms := r.requestsPerOpen[len(r.requestsPerOpen)*99/100] * msPerRequest
	avgRequests := float64(total) / float64(len(r.requestsPerOpen))

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
		consider(p.Name+" (partner daily limit)", svc.TruePartnerBookedTodayCents(id), p.DailyLimitCents)
	}
	overPct := float64(worstSpent-worstBudget) * 100 / float64(worstBudget)

	fmt.Printf("Simulated day: %d app opens on %d servers. Confirmations arrive late.\n", appOpens, servers)
	fmt.Printf("The shared counter service does not answer 1 in %d requests.\n\n", failEvery)

	ok := check("1. Each app open sends at most 1 request",
		total <= len(r.requestsPerOpen),
		fmt.Sprintf("Each app open sent %.1f requests on average.", avgRequests))
	ok = check("2. 99 of 100 app opens take 15 ms or less",
		p99ms <= 15,
		fmt.Sprintf("The slowest 1 in 100 app opens took %d ms.", p99ms)) && ok
	ok = check("3. No deal or partner spends more than 105% of its budget",
		overPct <= 5,
		fmt.Sprintf("%s spent $%.2f. Its budget is $%.2f. That is %.1f%% over.",
			worstName, float64(worstSpent)/100, float64(worstBudget)/100, overPct)) && ok
	ok = check("4. Affordable deals stay on the list",
		r.affordableButHidden == 0,
		fmt.Sprintf("%d app opens removed a deal that had used less than 80%% of its budget.", r.affordableButHidden)) && ok

	if r.firstBudgetUsedUpAt >= 0 {
		fmt.Printf("\nINFO   The first deal used its daily budget at %02d:%02d.\n", r.firstBudgetUsedUpAt/60, r.firstBudgetUsedUpAt%60)
	}
	return ok
}

func check(rule string, pass bool, detail string) bool {
	if pass {
		fmt.Printf("CHECK  %-58s PASS\n", rule)
	} else {
		fmt.Printf("CHECK  %-58s FAIL  %s\n", rule, detail)
	}
	return pass
}

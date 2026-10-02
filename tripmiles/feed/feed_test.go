package feed

import (
	"testing"

	"tripmiles/deals"
	"tripmiles/stats"
)

// exampleDeals are the 6 deals from the diagram in README.md.
func exampleDeals() *deals.Store {
	everywhere := []string{"DE", "US"}
	return deals.NewStore(
		[]deals.Deal{
			{ID: "skyline-lisbon", Name: "Skyline Air: Berlin to Lisbon", PartnerID: "skyline", Countries: everywhere,
				PriceCents: 400, CountryPriceCents: map[string]int64{"DE": 500}, DailyBudgetCents: 10000},
			{ID: "skyline-rome", Name: "Skyline Air: Berlin to Rome", PartnerID: "skyline", Countries: everywhere,
				PriceCents: 300, DailyBudgetCents: 50000},
			{ID: "harbor-paris", Name: "Harbor Hotels: Paris", PartnerID: "harbor", Countries: everywhere,
				PriceCents: 600, DailyBudgetCents: 50000},
			{ID: "harbor-vienna", Name: "Harbor Hotels: Vienna", PartnerID: "harbor", Countries: everywhere,
				PriceCents: 350, DailyBudgetCents: 50000},
			{ID: "nordic-oslo", Name: "Nordic Rail: Oslo to Bergen", PartnerID: "nordic", Countries: everywhere,
				PriceCents: 400, DailyBudgetCents: 50000},
			{ID: "nordic-stockholm", Name: "Nordic Rail: Stockholm to Oslo", PartnerID: "nordic", Countries: everywhere,
				PriceCents: 250, DailyBudgetCents: 50000},
		},
		[]deals.Partner{
			{ID: "skyline", Name: "Skyline Air"},
			{ID: "harbor", Name: "Harbor Hotels"},
			{ID: "nordic", Name: "Nordic Rail", DailyLimitCents: 15000},
		},
	)
}

func newFeed() (*Feed, *stats.Service) {
	svc := stats.NewService()
	return &Feed{Deals: exampleDeals(), Stats: svc.NewClient()}, svc
}

func shows(list []deals.Deal, id string) bool {
	for _, d := range list {
		if d.ID == id {
			return true
		}
	}
	return false
}

func book(f *Feed, id, country string, times int) {
	for _, d := range f.Deals.AvailableIn(country) {
		if d.ID == id {
			for i := 0; i < times; i++ {
				f.Book(d, country)
			}
		}
	}
}

func TestDealIsHiddenWhenItsBudgetIsUsedUp(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", "DE", 20) // 20 x $5 = $100
	svc.ConfirmAllButNewest(0)          // all confirmed
	if shows(f.List("DE"), "skyline-lisbon") {
		t.Fatal("Skyline Air: Berlin to Lisbon has spent its whole $100 budget, all confirmed, but was still shown.")
	}
}

func TestPartnerDealsAreHiddenWhenThePartnerLimitIsUsedUp(t *testing.T) {
	f, svc := newFeed()
	book(f, "nordic-oslo", "US", 40) // 40 x $4 = $160, over Nordic Rail's $150 daily limit
	svc.ConfirmAllButNewest(0)
	list := f.List("US")
	if shows(list, "nordic-oslo") || shows(list, "nordic-stockholm") {
		t.Fatal("Nordic Rail has used up its $150 daily limit, but its deals were still shown.")
	}
}

func TestBookingsThatAreNotConfirmedYetStillCount(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", "DE", 20) // 20 x $5 = $100
	svc.ConfirmAllButNewest(4)          // 16 confirmed ($80), 4 still waiting
	if shows(f.List("DE"), "skyline-lisbon") {
		t.Fatal("Skyline Air: Berlin to Lisbon has 20 bookings at the German price of $5, which is its whole $100 budget. " +
			"Only 16 are confirmed ($80) and 4 are still waiting, but every booking counts. The deal was still shown.")
	}
}

func TestOpeningTheAppAsksTheCounterServiceAtMostOnce(t *testing.T) {
	f, _ := newFeed()
	before := f.Stats.Calls()
	f.List("DE")
	if asked := f.Stats.Calls() - before; asked > 1 {
		t.Fatalf("Opening the app asked the shared counter service %d times. It should ask at most once.", asked)
	}
}

func TestDealOverBudgetIsHiddenWhenTheCounterServiceDoesNotAnswer(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", "DE", 20)
	svc.ConfirmAllButNewest(0)
	svc.FailEveryNth = 1 // from now on, no question gets an answer
	if shows(f.List("DE"), "skyline-lisbon") {
		t.Fatal("The shared counter service did not answer, and a deal that had used up its budget was still shown.")
	}
}

func TestAffordableDealsAreShown(t *testing.T) {
	f, _ := newFeed()
	if got := len(f.List("DE")); got != 6 {
		t.Fatalf("Nothing has been booked yet, so all 6 deals should be shown, but %d were.", got)
	}
}

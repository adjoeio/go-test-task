package feed

import (
	"testing"

	"tripmiles/deals"
	"tripmiles/stats"
)

// exampleDeals returns the 6 deals from the diagram in README.md.
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
	svc.ConfirmAllButNewest(0)          // The partner confirms all bookings.
	if shows(f.List("DE"), "skyline-lisbon") {
		t.Fatal("Skyline Air: Berlin to Lisbon has $100 of confirmed spend. Its daily budget is $100. The deal is still on the list.")
	}
}

func TestPartnerDealsAreHiddenWhenThePartnerLimitIsUsedUp(t *testing.T) {
	f, svc := newFeed()
	book(f, "nordic-oslo", "US", 40) // 40 x $4 = $160. The partner daily limit of Nordic Rail is $150.
	svc.ConfirmAllButNewest(0)
	list := f.List("US")
	if shows(list, "nordic-oslo") || shows(list, "nordic-stockholm") {
		t.Fatal("Nordic Rail has $160 of confirmed spend. Its partner daily limit is $150. Its deals are still on the list.")
	}
}

func TestBookingsThatAreNotConfirmedYetStillCount(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", "DE", 20) // 20 x $5 = $100
	svc.ConfirmAllButNewest(4)          // 16 bookings are confirmed ($80). 4 bookings are unconfirmed.
	if shows(f.List("DE"), "skyline-lisbon") {
		t.Fatal("Skyline Air: Berlin to Lisbon has 20 bookings at the German price of $5. That is $100, the full daily budget. " +
			"16 bookings are confirmed ($80). 4 bookings are unconfirmed ($20). All bookings count. The deal is still on the list.")
	}
}

func TestOpeningTheAppAsksTheCounterServiceAtMostOnce(t *testing.T) {
	f, _ := newFeed()
	before := f.Stats.Calls()
	f.List("DE")
	if asked := f.Stats.Calls() - before; asked > 1 {
		t.Fatalf("One app open sent %d requests to the shared counter service. The limit is 1 request.", asked)
	}
}

func TestDealOverBudgetIsHiddenWhenTheCounterServiceDoesNotAnswer(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", "DE", 20)
	svc.ConfirmAllButNewest(0)
	svc.FailEveryNth = 1 // From now on, the service does not answer any request.
	if shows(f.List("DE"), "skyline-lisbon") {
		t.Fatal("The shared counter service did not answer. Skyline Air: Berlin to Lisbon had used its $100 daily budget. The deal is still on the list.")
	}
}

func TestAffordableDealsAreShown(t *testing.T) {
	f, _ := newFeed()
	if got := len(f.List("DE")); got != 6 {
		t.Fatalf("There are no bookings yet. All 6 deals must be on the list. The list has %d deals.", got)
	}
}

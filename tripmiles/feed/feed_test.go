package feed

import (
	"testing"

	"tripmiles/deals"
	"tripmiles/dealsvc"
)

// exampleDeals returns the 6 deals from the diagram in README.md.
func exampleDeals() *deals.Store {
	return deals.NewStore(
		[]deals.Deal{
			{ID: "skyline-lisbon", Name: "Skyline Air: Berlin to Lisbon", PartnerID: "skyline", DailyLimit: 20},
			{ID: "skyline-rome", Name: "Skyline Air: Berlin to Rome", PartnerID: "skyline", DailyLimit: 100},
			{ID: "harbor-paris", Name: "Harbor Air: Hamburg to Paris", PartnerID: "harbor", DailyLimit: 100},
			{ID: "harbor-vienna", Name: "Harbor Air: Hamburg to Vienna", PartnerID: "harbor", DailyLimit: 100},
			{ID: "nordic-oslo", Name: "Nordic Air: Munich to Oslo", PartnerID: "nordic", DailyLimit: 100},
			{ID: "nordic-stockholm", Name: "Nordic Air: Munich to Stockholm", PartnerID: "nordic", DailyLimit: 100},
		},
		[]deals.Partner{
			{ID: "skyline", Name: "Skyline Air"},
			{ID: "harbor", Name: "Harbor Air"},
			{ID: "nordic", Name: "Nordic Air", DailyLimit: 30},
		},
	)
}

func newFeed() (*Feed, *dealsvc.Service) {
	svc := dealsvc.NewService()
	return &Feed{Deals: exampleDeals(), DealsSvc: svc.NewClient()}, svc
}

func shows(list []deals.Deal, id string) bool {
	for _, d := range list {
		if d.ID == id {
			return true
		}
	}
	return false
}

func book(f *Feed, id string, times int) {
	for _, d := range f.Deals.All() {
		if d.ID == id {
			for i := 0; i < times; i++ {
				f.Book(d)
			}
		}
	}
}

func TestDealIsHiddenWhenItsDailyLimitIsReached(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", 20)
	svc.ConfirmAllButNewest(0) // The partner confirms all bookings.
	if shows(f.List(), "skyline-lisbon") {
		t.Fatal("Skyline Air: Berlin to Lisbon has 20 confirmed bookings. Its daily limit is 20. The deal is still on the list.")
	}
}

func TestPartnerDealsAreHiddenWhenThePartnerLimitIsReached(t *testing.T) {
	f, svc := newFeed()
	book(f, "nordic-oslo", 30)
	svc.ConfirmAllButNewest(0)
	list := f.List()
	if shows(list, "nordic-oslo") || shows(list, "nordic-stockholm") {
		t.Fatal("Nordic Air has 30 confirmed bookings. Its partner daily limit is 30. Its deals are still on the list.")
	}
}

func TestBookingsThatAreNotConfirmedYetStillCount(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", 20)
	svc.ConfirmAllButNewest(4) // 16 bookings are confirmed. 4 bookings are unconfirmed.
	if shows(f.List(), "skyline-lisbon") {
		t.Fatal("Skyline Air: Berlin to Lisbon has 16 confirmed and 4 unconfirmed bookings. That is 20, its full daily limit. " +
			"All bookings count. The deal is still on the list.")
	}
}

func TestOpeningTheAppAsksTheDealsServiceAtMostOnce(t *testing.T) {
	f, _ := newFeed()
	before := f.DealsSvc.Calls()
	f.List()
	if asked := f.DealsSvc.Calls() - before; asked > 1 {
		t.Fatalf("One app open sent %d requests to the Deals Service. The limit is 1 request.", asked)
	}
}

func TestDealOverLimitIsHiddenWhenTheDealsServiceDoesNotAnswer(t *testing.T) {
	f, svc := newFeed()
	book(f, "skyline-lisbon", 20)
	svc.ConfirmAllButNewest(0)
	svc.FailEveryNth = 1 // From now on, the service does not answer any request.
	if shows(f.List(), "skyline-lisbon") {
		t.Fatal("The Deals Service did not answer. Skyline Air: Berlin to Lisbon had reached its daily limit of 20. The deal is still on the list.")
	}
}

func TestDealsWithStaysLeftAreShown(t *testing.T) {
	f, _ := newFeed()
	if got := len(f.List()); got != 6 {
		t.Fatalf("There are no bookings yet. All 6 deals must be on the list. The list has %d deals.", got)
	}
}

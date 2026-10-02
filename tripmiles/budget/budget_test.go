package budget

import (
	"testing"

	"tripmiles/deals"
)

func someDeals() []deals.Deal {
	return []deals.Deal{
		{ID: "rich", PricePerBookingMicros: 1_000_000, BudgetMicros: 10_000_000},
		{ID: "almost-empty", PricePerBookingMicros: 1_000_000, BudgetMicros: 10_000_000},
	}
}

func TestFilterHidesDealThatCannotAffordAnotherBooking(t *testing.T) {
	s := NewSpend()
	s.Charge("almost-empty", 9_500_000)
	got := Filter(s, someDeals())
	if len(got) != 1 || got[0].ID != "rich" {
		t.Fatalf("got %v, want only rich", got)
	}
}

func TestFilterUsesOneSpendLookupPerRequest(t *testing.T) {
	s := NewSpend()
	Filter(s, someDeals())
	if s.Calls() != 1 {
		t.Fatalf("spend lookups = %d, want 1", s.Calls())
	}
}

func TestFilterWhenSpendIsUnavailable(t *testing.T) {
	s := NewSpend()
	s.FailEvery = 1
	if got := Filter(s, someDeals()); len(got) != 0 {
		t.Fatalf("got %d deals while spend is unknown, want 0", len(got))
	}
}

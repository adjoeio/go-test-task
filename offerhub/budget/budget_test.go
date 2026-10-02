package budget

import (
	"testing"

	"offerhub/catalog"
)

func campaigns() []catalog.Campaign {
	return []catalog.Campaign{
		{ID: "rich", CPIMicros: 1_000_000, BudgetMicros: 10_000_000},
		{ID: "almost-empty", CPIMicros: 1_000_000, BudgetMicros: 10_000_000},
	}
}

func TestFilterHidesCampaignThatCannotAffordAnotherInstall(t *testing.T) {
	s := NewSpend()
	s.Charge("almost-empty", 9_500_000)
	got := Filter(s, campaigns())
	if len(got) != 1 || got[0].ID != "rich" {
		t.Fatalf("got %v, want only rich", got)
	}
}

func TestFilterUsesOneSpendLookupPerRequest(t *testing.T) {
	s := NewSpend()
	Filter(s, campaigns())
	if s.Calls() != 1 {
		t.Fatalf("spend lookups = %d, want 1", s.Calls())
	}
}

func TestFilterWhenSpendIsUnavailable(t *testing.T) {
	s := NewSpend()
	s.FailEvery = 1
	if got := Filter(s, campaigns()); len(got) != 0 {
		t.Fatalf("got %d campaigns while spend is unknown, want 0", len(got))
	}
}

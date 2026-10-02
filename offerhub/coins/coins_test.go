package coins

import "testing"

func TestCoins(t *testing.T) {
	// $2.00 CPI, 30% margin, publisher pays 1000 coins per USD -> 1400 coins.
	if got := Coins(2_000_000, 30, 1000); got != 1400 {
		t.Fatalf("got %d, want 1400", got)
	}
	if got := Coins(100, 30, 1000); got != 1 {
		t.Fatalf("got %d, want minimum of 1", got)
	}
}

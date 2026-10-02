package miles

import "testing"

func TestMiles(t *testing.T) {
	// $2.00 price, 30% fee, 1000 miles per USD -> 1400 miles.
	if got := Miles(2_000_000, 30, 1000); got != 1400 {
		t.Fatalf("got %d, want 1400", got)
	}
	if got := Miles(100, 30, 1000); got != 1 {
		t.Fatalf("got %d, want minimum of 1", got)
	}
}

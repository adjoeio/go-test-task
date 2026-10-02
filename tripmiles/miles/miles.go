// Package miles turns a partner's price into the miles a traveller sees.
package miles

// Miles = price × (1 − fee) × miles per USD, never below 1.
// Money is integer micro-USD so rounding is explicit.
func Miles(priceMicros, feePct, milesPerUSD int64) int64 {
	m := priceMicros * (100 - feePct) / 100 * milesPerUSD / 1_000_000
	if m < 1 {
		return 1
	}
	return m
}

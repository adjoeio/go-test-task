// Package coins turns an advertiser's price into the coins a user sees.
package coins

// Coins = CPI × (1 − margin) × coinsPerUSD, never below 1.
// Money is integer micro-USD so rounding is explicit.
func Coins(cpiMicros, marginPct, coinsPerUSD int64) int64 {
	c := cpiMicros * (100 - marginPct) / 100 * coinsPerUSD / 1_000_000
	if c < 1 {
		return 1
	}
	return c
}

package history

import (
	"math"
	"time"
)

// GenerateHistory produces 24 months of deterministic mock historical flight prices
// for the given origin/destination pair, ordered from oldest to most recent.
// Prices are derived from the IATA codes so the same route always returns the
// same values, and vary by season to reflect realistic demand patterns.
func GenerateHistory(origin, destination string) []MonthlyAverage {
	base := basePrice(origin, destination)
	now := time.Now()

	history := make([]MonthlyAverage, 24)
	for i := 0; i < 24; i++ {
		// Walk from 23 months ago (i=0) up to the current month (i=23).
		t := now.AddDate(0, -(23 - i), 0)
		avg := math.Round(base*seasonMultiplier(t.Month())*100) / 100
		history[i] = MonthlyAverage{
			Year:     t.Year(),
			Month:    int(t.Month()),
			AvgPrice: avg,
			Currency: "BRL",
		}
	}
	return history
}

// basePrice derives a stable base price in the range [1500, 6000] from the IATA codes.
func basePrice(origin, destination string) float64 {
	sum := 0
	for _, c := range origin + destination {
		sum += int(c)
	}
	return 1500.0 + float64(sum%4501)
}

// seasonMultiplier returns a price adjustment factor that reflects typical
// demand patterns observed in the Brazilian aviation market.
func seasonMultiplier(m time.Month) float64 {
	switch m {
	case time.December, time.January:
		return 1.30 // peak holiday season
	case time.July:
		return 1.20 // southern-hemisphere winter school break
	case time.June, time.August:
		return 1.10
	case time.March, time.April:
		return 0.90 // post-carnival shoulder season
	default:
		return 1.00
	}
}

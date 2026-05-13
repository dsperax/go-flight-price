package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/sperax/flight-price-service/internal/flights"
)

// Amadeus is a mock adapter for the Amadeus flight API.
// To use the real API, replace the Search method body with an HTTP call
// using the AMADEUS_API_KEY environment variable.
type Amadeus struct {
	apiKey string
}

func NewAmadeus(apiKey string) *Amadeus {
	return &Amadeus{apiKey: apiKey}
}

func (a *Amadeus) Name() string {
	return "Amadeus"
}

func (a *Amadeus) Search(ctx context.Context, req flights.FlightSearchRequest) ([]flights.Flight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	date, _ := time.Parse("2006-01-02", req.Date)

	return []flights.Flight{
		{
			Provider:        a.Name(),
			Airline:         "LATAM",
			FlightNumber:    fmt.Sprintf("LA%s%s", req.Origin, req.Destination),
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   time.Date(date.Year(), date.Month(), date.Day(), 22, 50, 0, 0, time.UTC),
			ArrivalTime:     time.Date(date.Year(), date.Month(), date.Day()+1, 7, 15, 0, 0, time.UTC),
			DurationMinutes: 625,
			Price:           3250.90,
			Currency:        "BRL",
		},
		{
			Provider:        a.Name(),
			Airline:         "Gol",
			FlightNumber:    fmt.Sprintf("G3%s%s", req.Origin, req.Destination),
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   time.Date(date.Year(), date.Month(), date.Day(), 18, 30, 0, 0, time.UTC),
			ArrivalTime:     time.Date(date.Year(), date.Month(), date.Day()+1, 3, 20, 0, 0, time.UTC),
			DurationMinutes: 650,
			Price:           2980.00,
			Currency:        "BRL",
		},
	}, nil
}

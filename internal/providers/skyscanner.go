package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/sperax/flight-price-service/internal/flights"
)

// Skyscanner is a mock adapter for the Skyscanner flight API.
// To use the real API, replace the Search method body with an HTTP call
// using the SKYSCANNER_API_KEY environment variable.
type Skyscanner struct {
	apiKey string
}

func NewSkyscanner(apiKey string) *Skyscanner {
	return &Skyscanner{apiKey: apiKey}
}

func (s *Skyscanner) Name() string {
	return "Skyscanner"
}

func (s *Skyscanner) Search(ctx context.Context, req flights.FlightSearchRequest) ([]flights.Flight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	date, _ := time.Parse("2006-01-02", req.Date)

	return []flights.Flight{
		{
			Provider:        s.Name(),
			Airline:         "Delta",
			FlightNumber:    fmt.Sprintf("DL%s%s", req.Origin, req.Destination),
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   time.Date(date.Year(), date.Month(), date.Day(), 21, 30, 0, 0, time.UTC),
			ArrivalTime:     time.Date(date.Year(), date.Month(), date.Day()+1, 5, 35, 0, 0, time.UTC),
			DurationMinutes: 565,
			Price:           4100.00,
			Currency:        "BRL",
		},
		{
			Provider:        s.Name(),
			Airline:         "American Airlines",
			FlightNumber:    fmt.Sprintf("AA%s%s", req.Origin, req.Destination),
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   time.Date(date.Year(), date.Month(), date.Day(), 23, 55, 0, 0, time.UTC),
			ArrivalTime:     time.Date(date.Year(), date.Month(), date.Day()+1, 8, 40, 0, 0, time.UTC),
			DurationMinutes: 585,
			Price:           3750.50,
			Currency:        "BRL",
		},
	}, nil
}

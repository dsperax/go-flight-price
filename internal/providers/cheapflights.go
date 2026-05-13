package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/sperax/flight-price-service/internal/flights"
)

// CheapFlights is a mock adapter for the CheapFlights API.
// To use the real API, replace the Search method body with an HTTP call
// using the CHEAPFLIGHTS_API_KEY environment variable.
type CheapFlights struct {
	apiKey string
}

func NewCheapFlights(apiKey string) *CheapFlights {
	return &CheapFlights{apiKey: apiKey}
}

func (c *CheapFlights) Name() string {
	return "CheapFlights"
}

func (c *CheapFlights) Search(ctx context.Context, req flights.FlightSearchRequest) ([]flights.Flight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	date, _ := time.Parse("2006-01-02", req.Date)

	return []flights.Flight{
		{
			Provider:        c.Name(),
			Airline:         "Azul",
			FlightNumber:    fmt.Sprintf("AD%s%s", req.Origin, req.Destination),
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   time.Date(date.Year(), date.Month(), date.Day(), 16, 45, 0, 0, time.UTC),
			ArrivalTime:     time.Date(date.Year(), date.Month(), date.Day()+1, 2, 55, 0, 0, time.UTC),
			DurationMinutes: 610,
			Price:           2750.00,
			Currency:        "BRL",
		},
		{
			Provider:        c.Name(),
			Airline:         "United",
			FlightNumber:    fmt.Sprintf("UA%s%s", req.Origin, req.Destination),
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   time.Date(date.Year(), date.Month(), date.Day(), 20, 10, 0, 0, time.UTC),
			ArrivalTime:     time.Date(date.Year(), date.Month(), date.Day()+1, 6, 30, 0, 0, time.UTC),
			DurationMinutes: 620,
			Price:           3100.75,
			Currency:        "BRL",
		},
	}, nil
}

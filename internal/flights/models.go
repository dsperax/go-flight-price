package flights

import (
	"errors"
	"time"
)

// Flight represents a single flight option returned by a provider.
type Flight struct {
	Provider        string    `json:"provider"`
	Airline         string    `json:"airline"`
	FlightNumber    string    `json:"flight_number"`
	Origin          string    `json:"origin"`
	Destination     string    `json:"destination"`
	DepartureTime   time.Time `json:"departure_time"`
	ArrivalTime     time.Time `json:"arrival_time"`
	DurationMinutes int       `json:"duration_minutes"`
	Price           float64   `json:"price"`
	Currency        string    `json:"currency"`
}

// FlightSearchRequest holds the validated parameters for a flight search.
type FlightSearchRequest struct {
	Origin      string
	Destination string
	Date        string
}

// Validate checks that all required fields are present and well-formed.
func (r FlightSearchRequest) Validate() error {
	if r.Origin == "" {
		return errors.New("origin is required")
	}
	if r.Destination == "" {
		return errors.New("destination is required")
	}
	if r.Date == "" {
		return errors.New("date is required")
	}
	if _, err := time.Parse("2006-01-02", r.Date); err != nil {
		return errors.New("date must use YYYY-MM-DD format")
	}
	return nil
}

// ProviderError describes a failure from a single provider.
type ProviderError struct {
	Provider string `json:"provider"`
	Message  string `json:"message"`
}

// FlightSearchResponse is the aggregated result returned to the client.
type FlightSearchResponse struct {
	Origin         string          `json:"origin"`
	Destination    string          `json:"destination"`
	Date           string          `json:"date"`
	CheapestFlight *Flight         `json:"cheapest_flight"`
	FastestFlight  *Flight         `json:"fastest_flight"`
	Flights        []Flight        `json:"flights"`
	ProviderErrors []ProviderError `json:"provider_errors"`
	Cached         bool            `json:"cached"`
}

package flights_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sperax/flight-price-service/internal/flights"
)

// --- stub providers ---

type stubProvider struct {
	name    string
	flights []flights.Flight
	err     error
	delay   time.Duration
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) Search(ctx context.Context, _ flights.FlightSearchRequest) ([]flights.Flight, error) {
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.flights, s.err
}

// helper to create flights with given price and duration.
func makeFlight(provider string, price float64, duration int) flights.Flight {
	return flights.Flight{
		Provider:        provider,
		Airline:         "TestAir",
		FlightNumber:    "TA001",
		Origin:          "GRU",
		Destination:     "JFK",
		DepartureTime:   time.Now(),
		ArrivalTime:     time.Now().Add(time.Duration(duration) * time.Minute),
		DurationMinutes: duration,
		Price:           price,
		Currency:        "BRL",
	}
}

var validReq = flights.FlightSearchRequest{
	Origin:      "GRU",
	Destination: "JFK",
	Date:        "2026-06-10",
}

// --- tests ---

func TestService_AllProvidersSucceed(t *testing.T) {
	p1 := &stubProvider{name: "P1", flights: []flights.Flight{makeFlight("P1", 3000, 600)}}
	p2 := &stubProvider{name: "P2", flights: []flights.Flight{makeFlight("P2", 2500, 650)}}
	p3 := &stubProvider{name: "P3", flights: []flights.Flight{makeFlight("P3", 4000, 500)}}

	svc := flights.NewService([]flights.FlightProvider{p1, p2, p3}, 3)
	resp, err := svc.Search(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Flights) != 3 {
		t.Errorf("expected 3 flights, got %d", len(resp.Flights))
	}
	if len(resp.ProviderErrors) != 0 {
		t.Errorf("expected no provider errors, got %d", len(resp.ProviderErrors))
	}
}

func TestService_AllProvidersFail(t *testing.T) {
	p1 := &stubProvider{name: "P1", err: errors.New("timeout")}
	p2 := &stubProvider{name: "P2", err: errors.New("auth error")}

	svc := flights.NewService([]flights.FlightProvider{p1, p2}, 3)
	_, err := svc.Search(context.Background(), validReq)

	if !errors.Is(err, flights.ErrAllProvidersFailed) {
		t.Errorf("expected ErrAllProvidersFailed, got %v", err)
	}
}

func TestService_PartialProviderFailure(t *testing.T) {
	p1 := &stubProvider{name: "P1", flights: []flights.Flight{makeFlight("P1", 3000, 600)}}
	p2 := &stubProvider{name: "P2", err: errors.New("service unavailable")}

	svc := flights.NewService([]flights.FlightProvider{p1, p2}, 3)
	resp, err := svc.Search(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Flights) != 1 {
		t.Errorf("expected 1 flight, got %d", len(resp.Flights))
	}
	if len(resp.ProviderErrors) != 1 {
		t.Errorf("expected 1 provider error, got %d", len(resp.ProviderErrors))
	}
	if resp.ProviderErrors[0].Provider != "P2" {
		t.Errorf("expected provider error from P2, got %s", resp.ProviderErrors[0].Provider)
	}
}

func TestService_ProviderTimeout(t *testing.T) {
	slow := &stubProvider{name: "Slow", delay: 5 * time.Second, flights: []flights.Flight{makeFlight("Slow", 2000, 400)}}
	fast := &stubProvider{name: "Fast", flights: []flights.Flight{makeFlight("Fast", 3000, 500)}}

	// 100ms timeout — slow provider should be cut off
	svc := flights.NewService([]flights.FlightProvider{slow, fast}, 0)
	// Use a tiny timeout via context instead of the service timeout to keep the test fast
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resp, err := svc.Search(ctx, validReq)

	// Either the slow provider timed out (partial success) or both did (all failed)
	if err != nil && !errors.Is(err, flights.ErrAllProvidersFailed) {
		t.Fatalf("unexpected error type: %v", err)
	}
	if err == nil {
		// Fast provider must have replied
		if len(resp.Flights) == 0 {
			t.Error("expected at least one flight from fast provider")
		}
	}
}

func TestService_CheapestAndFastestSelected(t *testing.T) {
	cheap := makeFlight("P1", 1000, 700) // cheapest
	fast := makeFlight("P2", 5000, 300)  // fastest
	mid := makeFlight("P3", 3000, 500)

	p := &stubProvider{name: "P1", flights: []flights.Flight{cheap, fast, mid}}

	svc := flights.NewService([]flights.FlightProvider{p}, 3)
	resp, err := svc.Search(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CheapestFlight == nil || resp.CheapestFlight.Price != 1000 {
		t.Errorf("expected cheapest price 1000, got %v", resp.CheapestFlight)
	}
	if resp.FastestFlight == nil || resp.FastestFlight.DurationMinutes != 300 {
		t.Errorf("expected fastest duration 300, got %v", resp.FastestFlight)
	}
}

func TestService_ResponseMetadata(t *testing.T) {
	p := &stubProvider{name: "P1", flights: []flights.Flight{makeFlight("P1", 2000, 500)}}

	svc := flights.NewService([]flights.FlightProvider{p}, 3)
	resp, err := svc.Search(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Origin != validReq.Origin {
		t.Errorf("expected origin %s, got %s", validReq.Origin, resp.Origin)
	}
	if resp.Destination != validReq.Destination {
		t.Errorf("expected destination %s, got %s", validReq.Destination, resp.Destination)
	}
	if resp.Date != validReq.Date {
		t.Errorf("expected date %s, got %s", validReq.Date, resp.Date)
	}
	if resp.Cached {
		t.Error("expected cached=false for fresh search")
	}
}

func TestService_FlightsSortedByPriceAsc(t *testing.T) {
	list := []flights.Flight{
		makeFlight("P1", 5000, 400),
		makeFlight("P1", 1500, 600),
		makeFlight("P1", 3000, 550),
	}
	p := &stubProvider{name: "P1", flights: list}

	svc := flights.NewService([]flights.FlightProvider{p}, 3)
	resp, err := svc.Search(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Flights) != 3 {
		t.Fatalf("expected 3 flights, got %d", len(resp.Flights))
	}
	for i := 1; i < len(resp.Flights); i++ {
		if resp.Flights[i].Price < resp.Flights[i-1].Price {
			t.Errorf("flights not sorted by price: index %d (%.2f) < index %d (%.2f)",
				i, resp.Flights[i].Price, i-1, resp.Flights[i-1].Price)
		}
	}
}

func TestService_SortTieBreakByDuration(t *testing.T) {
	list := []flights.Flight{
		makeFlight("P1", 2000, 700),
		makeFlight("P1", 2000, 400),
		makeFlight("P1", 2000, 550),
	}
	p := &stubProvider{name: "P1", flights: list}

	svc := flights.NewService([]flights.FlightProvider{p}, 3)
	resp, err := svc.Search(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i < len(resp.Flights); i++ {
		if resp.Flights[i].DurationMinutes < resp.Flights[i-1].DurationMinutes {
			t.Errorf("tie-break not sorted by duration: index %d (%d min) < index %d (%d min)",
				i, resp.Flights[i].DurationMinutes, i-1, resp.Flights[i-1].DurationMinutes)
		}
	}
}

func TestService_CheapestIsFirstAfterSort(t *testing.T) {
	p := &stubProvider{name: "P1", flights: []flights.Flight{
		makeFlight("P1", 4000, 300),
		makeFlight("P1", 900, 700),
		makeFlight("P1", 2500, 500),
	}}

	svc := flights.NewService([]flights.FlightProvider{p}, 3)
	resp, err := svc.Search(context.Background(), validReq)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CheapestFlight == nil {
		t.Fatal("expected cheapest_flight to be set")
	}
	if resp.CheapestFlight.Price != 900 {
		t.Errorf("expected cheapest price 900, got %.2f", resp.CheapestFlight.Price)
	}
	// After sorting, cheapest must be the first element in Flights.
	if resp.Flights[0].Price != resp.CheapestFlight.Price {
		t.Errorf("cheapest_flight price %.2f does not match first sorted flight %.2f",
			resp.CheapestFlight.Price, resp.Flights[0].Price)
	}
}

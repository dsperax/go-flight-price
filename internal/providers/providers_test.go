package providers_test

import (
	"context"
	"testing"

	"github.com/sperax/flight-price-service/internal/flights"
	"github.com/sperax/flight-price-service/internal/providers"
)

var validReq = flights.FlightSearchRequest{
	Origin:      "GRU",
	Destination: "JFK",
	Date:        "2026-06-10",
}

func assertProviderResults(t *testing.T, p flights.FlightProvider, req flights.FlightSearchRequest) {
	t.Helper()

	result, err := p.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("[%s] unexpected error: %v", p.Name(), err)
	}
	if len(result) == 0 {
		t.Fatalf("[%s] expected at least one flight, got 0", p.Name())
	}

	for i, f := range result {
		if f.Provider != p.Name() {
			t.Errorf("[%s] flight %d: expected provider %q, got %q", p.Name(), i, p.Name(), f.Provider)
		}
		if f.Origin != req.Origin {
			t.Errorf("[%s] flight %d: expected origin %q, got %q", p.Name(), i, req.Origin, f.Origin)
		}
		if f.Destination != req.Destination {
			t.Errorf("[%s] flight %d: expected destination %q, got %q", p.Name(), i, req.Destination, f.Destination)
		}
		if f.Price <= 0 {
			t.Errorf("[%s] flight %d: expected positive price, got %f", p.Name(), i, f.Price)
		}
		if f.DurationMinutes <= 0 {
			t.Errorf("[%s] flight %d: expected positive duration, got %d", p.Name(), i, f.DurationMinutes)
		}
		if f.Currency == "" {
			t.Errorf("[%s] flight %d: expected non-empty currency", p.Name(), i)
		}
		if f.ArrivalTime.Before(f.DepartureTime) {
			t.Errorf("[%s] flight %d: arrival is before departure", p.Name(), i)
		}
	}
}

func TestAmadeus_Search(t *testing.T) {
	p := providers.NewAmadeus("")
	if p.Name() != "Amadeus" {
		t.Errorf("expected name Amadeus, got %s", p.Name())
	}
	assertProviderResults(t, p, validReq)
}

func TestSkyscanner_Search(t *testing.T) {
	p := providers.NewSkyscanner("")
	if p.Name() != "Skyscanner" {
		t.Errorf("expected name Skyscanner, got %s", p.Name())
	}
	assertProviderResults(t, p, validReq)
}

func TestCheapFlights_Search(t *testing.T) {
	p := providers.NewCheapFlights("")
	if p.Name() != "CheapFlights" {
		t.Errorf("expected name CheapFlights, got %s", p.Name())
	}
	assertProviderResults(t, p, validReq)
}

func TestProviders_ReflectRequestData(t *testing.T) {
	// Verify that providers use the request fields, not hardcoded values.
	req := flights.FlightSearchRequest{
		Origin:      "CGH",
		Destination: "LHR",
		Date:        "2026-09-01",
	}

	providerList := []flights.FlightProvider{
		providers.NewAmadeus(""),
		providers.NewSkyscanner(""),
		providers.NewCheapFlights(""),
	}

	for _, p := range providerList {
		result, err := p.Search(context.Background(), req)
		if err != nil {
			t.Fatalf("[%s] unexpected error: %v", p.Name(), err)
		}
		for i, f := range result {
			if f.Origin != "CGH" {
				t.Errorf("[%s] flight %d: expected origin CGH, got %s", p.Name(), i, f.Origin)
			}
			if f.Destination != "LHR" {
				t.Errorf("[%s] flight %d: expected destination LHR, got %s", p.Name(), i, f.Destination)
			}
			if f.DepartureTime.Year() != 2026 {
				t.Errorf("[%s] flight %d: expected year 2026, got %d", p.Name(), i, f.DepartureTime.Year())
			}
		}
	}
}

func TestProviders_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	providerList := []flights.FlightProvider{
		providers.NewAmadeus(""),
		providers.NewSkyscanner(""),
		providers.NewCheapFlights(""),
	}

	for _, p := range providerList {
		_, err := p.Search(ctx, validReq)
		if err == nil {
			t.Errorf("[%s] expected error for cancelled context, got nil", p.Name())
		}
	}
}

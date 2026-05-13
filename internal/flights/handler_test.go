package flights_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sperax/flight-price-service/internal/flights"
)

// handlerStubProvider is a local stub used only in handler tests.
type handlerStubProvider struct {
	name    string
	results []flights.Flight
	err     error
}

func (h *handlerStubProvider) Name() string { return h.name }
func (h *handlerStubProvider) Search(_ context.Context, _ flights.FlightSearchRequest) ([]flights.Flight, error) {
	return h.results, h.err
}

func makeHandlerFlight(price float64, duration int) flights.Flight {
	return flights.Flight{
		Provider:        "Test",
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

func newTestHandler(p flights.FlightProvider) *flights.Handler {
	svc := flights.NewService([]flights.FlightProvider{p}, 3, 0)
	return flights.NewHandler(svc)
}

func TestHandler_Search_MissingOrigin(t *testing.T) {
	h := newTestHandler(&handlerStubProvider{name: "P1"})

	req := httptest.NewRequest(http.MethodGet, "/flights/search?destination=JFK&date=2026-06-10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %s", body["error"])
	}
}

func TestHandler_Search_MissingDestination(t *testing.T) {
	h := newTestHandler(&handlerStubProvider{name: "P1"})

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&date=2026-06-10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_Search_MissingDate(t *testing.T) {
	h := newTestHandler(&handlerStubProvider{name: "P1"})

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_Search_InvalidDateFormat(t *testing.T) {
	h := newTestHandler(&handlerStubProvider{name: "P1"})

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK&date=10-06-2026", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_Search_AllProvidersFail(t *testing.T) {
	p := &handlerStubProvider{name: "P1", err: errors.New("unavailable")}
	h := newTestHandler(p)

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rr.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body["error"] != "provider_unavailable" {
		t.Errorf("expected error=provider_unavailable, got %s", body["error"])
	}
}

func TestHandler_Search_Success(t *testing.T) {
	p := &handlerStubProvider{
		name:    "P1",
		results: []flights.Flight{makeHandlerFlight(3000, 600), makeHandlerFlight(2000, 650)},
	}
	h := newTestHandler(p)

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var resp flights.FlightSearchResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Origin != "GRU" {
		t.Errorf("expected origin GRU, got %s", resp.Origin)
	}
	if resp.Destination != "JFK" {
		t.Errorf("expected destination JFK, got %s", resp.Destination)
	}
	if len(resp.Flights) != 2 {
		t.Errorf("expected 2 flights, got %d", len(resp.Flights))
	}
	if resp.CheapestFlight == nil {
		t.Error("expected cheapest_flight to be set")
	}
	if resp.FastestFlight == nil {
		t.Error("expected fastest_flight to be set")
	}
}

func TestHandler_Search_ContentTypeJSON(t *testing.T) {
	p := &handlerStubProvider{
		name:    "P1",
		results: []flights.Flight{makeHandlerFlight(3000, 600)},
	}
	h := newTestHandler(p)

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

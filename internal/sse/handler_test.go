package sse_test

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sperax/flight-price-service/internal/flights"
	"github.com/sperax/flight-price-service/internal/sse"
)

// --- test infrastructure ---

// stubSearcher implements sse.Searcher for unit tests.
type stubSearcher struct {
	resp flights.FlightSearchResponse
	err  error
}

func (s *stubSearcher) Search(_ context.Context, req flights.FlightSearchRequest) (flights.FlightSearchResponse, error) {
	res := s.resp
	res.Origin = req.Origin
	res.Destination = req.Destination
	return res, s.err
}

// sampleResponse returns a minimal but valid FlightSearchResponse.
func sampleResponse() flights.FlightSearchResponse {
	now := time.Now()
	f := flights.Flight{
		Provider: "TestAir", Airline: "TestAir", FlightNumber: "TA001",
		Origin: "GRU", Destination: "JFK",
		DepartureTime: now, ArrivalTime: now.Add(10 * time.Hour),
		DurationMinutes: 600, Price: 2500.00, Currency: "BRL",
	}
	return flights.FlightSearchResponse{
		Origin: "GRU", Destination: "JFK", Date: "2026-06-10",
		Flights:        []flights.Flight{f},
		CheapestFlight: &f,
		FastestFlight:  &f,
	}
}

// mountRouter wraps the SSE handler in a chi router to populate URL params,
// mirroring how server.go mounts it in production.
func mountRouter(h *sse.Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/subscribe/{route}", h.Subscribe)
	return r
}

// cancelledCtx returns a request whose context is already cancelled so the
// SSE handler exits immediately after emitting the initial event.
func cancelledCtx(method, target string) *http.Request {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return httptest.NewRequest(method, target, nil).WithContext(ctx)
}

// --- tests ---

func TestSSE_InvalidRouteFormat(t *testing.T) {
	h := sse.New(&stubSearcher{resp: sampleResponse(), err: nil})
	router := mountRouter(h)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/subscribe/INVALIDROUTE", nil)
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid route format, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "invalid_request") {
		t.Errorf("expected error=invalid_request in body, got: %s", w.Body.String())
	}
}

func TestSSE_SSEHeaders(t *testing.T) {
	h := sse.New(&stubSearcher{resp: sampleResponse()})

	router := mountRouter(h)

	w := httptest.NewRecorder()
	r := cancelledCtx(http.MethodGet, "/subscribe/GRU-JFK")
	router.ServeHTTP(w, r)

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cc)
	}
}

func TestSSE_InitialEventWritten(t *testing.T) {
	h := sse.New(&stubSearcher{resp: sampleResponse()})
	router := mountRouter(h)

	w := httptest.NewRecorder()
	r := cancelledCtx(http.MethodGet, "/subscribe/GRU-JFK")
	router.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.HasPrefix(body, "data: ") {
		t.Errorf("expected body to start with 'data: ', got: %s", body)
	}
	if !strings.Contains(body, "GRU") {
		t.Errorf("expected body to contain origin GRU, got: %s", body)
	}
}

func TestSSE_SearchError_NoEventEmitted(t *testing.T) {
	h := sse.New(&stubSearcher{err: errors.New("all providers failed"), resp: flights.FlightSearchResponse{}})

	router := mountRouter(h)

	w := httptest.NewRecorder()
	r := cancelledCtx(http.MethodGet, "/subscribe/GRU-JFK")
	router.ServeHTTP(w, r)

	if body := w.Body.String(); body != "" {
		t.Errorf("expected empty body when search fails, got: %s", body)
	}
}

func TestSSE_EventIsValidSSEFormat(t *testing.T) {
	h := sse.New(&stubSearcher{resp: sampleResponse()})
	router := mountRouter(h)

	w := httptest.NewRecorder()
	r := cancelledCtx(http.MethodGet, "/subscribe/GRU-JFK")
	router.ServeHTTP(w, r)

	scanner := bufio.NewScanner(strings.NewReader(w.Body.String()))
	foundData := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			foundData = true
			// The data field must be followed by an empty line (SSE spec).
			if !scanner.Scan() || scanner.Text() != "" {
				t.Error("expected empty line after 'data:' field (SSE spec)")
			}
			break
		}
	}
	if !foundData {
		t.Error("expected at least one 'data: ' line in SSE stream")
	}
}

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

// --- helpers shared across error-handling tests ---

type errHandlingProvider struct {
	name    string
	results []flights.Flight
	err     error
}

func (p *errHandlingProvider) Name() string { return p.name }
func (p *errHandlingProvider) Search(_ context.Context, _ flights.FlightSearchRequest) ([]flights.Flight, error) {
	return p.results, p.err
}

type panicProvider struct{ name string }

func (p *panicProvider) Name() string { return p.name }
func (p *panicProvider) Search(_ context.Context, _ flights.FlightSearchRequest) ([]flights.Flight, error) {
	panic("simulated provider panic")
}

func newErrHandlingSvc(p flights.FlightProvider) *flights.Service {
	return flights.NewService([]flights.FlightProvider{p}, 3, 0)
}

// --- ValidationError tests ---

func TestValidationError_Format(t *testing.T) {
	err := &flights.ValidationError{Field: "origin", Message: "is required"}
	want := "origin: is required"
	if err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

func TestValidationError_NoField(t *testing.T) {
	err := &flights.ValidationError{Message: "something went wrong"}
	if err.Error() != "something went wrong" {
		t.Errorf("unexpected: %s", err.Error())
	}
}

func TestValidate_ReturnsValidationError(t *testing.T) {
	req := flights.FlightSearchRequest{Origin: "", Destination: "JFK", Date: "2026-06-10"}
	err := req.Validate()

	var valErr *flights.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if valErr.Field != "origin" {
		t.Errorf("expected field=origin, got %s", valErr.Field)
	}
}

func TestValidate_DateFieldError(t *testing.T) {
	req := flights.FlightSearchRequest{Origin: "GRU", Destination: "JFK", Date: "13/06/2026"}
	err := req.Validate()

	var valErr *flights.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if valErr.Field != "date" {
		t.Errorf("expected field=date, got %s", valErr.Field)
	}
}

// --- Normalize tests ---

func TestNormalize_UppercasesIATA(t *testing.T) {
	req := flights.FlightSearchRequest{Origin: "gru", Destination: "jfk", Date: " 2026-06-10 "}
	req.Normalize()

	if req.Origin != "GRU" {
		t.Errorf("expected GRU, got %s", req.Origin)
	}
	if req.Destination != "JFK" {
		t.Errorf("expected JFK, got %s", req.Destination)
	}
	if req.Date != "2026-06-10" {
		t.Errorf("expected trimmed date, got %q", req.Date)
	}
}

func TestNormalize_TrimsSpaces(t *testing.T) {
	req := flights.FlightSearchRequest{Origin: "  GRU  ", Destination: "  JFK  ", Date: "2026-06-10"}
	req.Normalize()

	if req.Origin != "GRU" {
		t.Errorf("expected GRU, got %q", req.Origin)
	}
	if req.Destination != "JFK" {
		t.Errorf("expected JFK, got %q", req.Destination)
	}
}

// --- AllProvidersFailedError tests ---

func TestAllProvidersFailedError_IsErrAllProvidersFailed(t *testing.T) {
	err := &flights.AllProvidersFailedError{
		ProviderErrors: []flights.ProviderError{{Provider: "P1", Message: "timeout"}},
	}
	if !errors.Is(err, flights.ErrAllProvidersFailed) {
		t.Error("expected errors.Is to match ErrAllProvidersFailed")
	}
}

func TestAllProvidersFailedError_CarriesProviderErrors(t *testing.T) {
	p1 := &errHandlingProvider{name: "P1", err: errors.New("connection refused")}
	p2 := &errHandlingProvider{name: "P2", err: errors.New("auth failed")}

	svc := flights.NewService([]flights.FlightProvider{p1, p2}, 3, 0)
	_, err := svc.Search(context.Background(), flights.FlightSearchRequest{
		Origin: "GRU", Destination: "JFK", Date: "2026-06-10",
	})

	var apfErr *flights.AllProvidersFailedError
	if !errors.As(err, &apfErr) {
		t.Fatalf("expected *AllProvidersFailedError, got %T", err)
	}
	if len(apfErr.ProviderErrors) != 2 {
		t.Errorf("expected 2 provider errors, got %d", len(apfErr.ProviderErrors))
	}
}

// --- Panic recovery tests ---

func TestService_PanicRecovery_DoesNotCrash(t *testing.T) {
	good := &errHandlingProvider{
		name:    "Good",
		results: []flights.Flight{makeErrHandlingFlight(3000, 500)},
	}
	bad := &panicProvider{name: "Panicky"}

	svc := flights.NewService([]flights.FlightProvider{good, bad}, 3, 0)
	resp, err := svc.Search(context.Background(), flights.FlightSearchRequest{
		Origin: "GRU", Destination: "JFK", Date: "2026-06-10",
	})

	if err != nil {
		t.Fatalf("expected partial success, got error: %v", err)
	}
	if len(resp.ProviderErrors) != 1 {
		t.Errorf("expected 1 provider error (from panic), got %d", len(resp.ProviderErrors))
	}
	if len(resp.Flights) != 1 {
		t.Errorf("expected 1 flight from good provider, got %d", len(resp.Flights))
	}
}

func TestService_PanicRecovery_AllPanic_ReturnsError(t *testing.T) {
	bad := &panicProvider{name: "Panicky"}

	svc := flights.NewService([]flights.FlightProvider{bad}, 3, 0)
	_, err := svc.Search(context.Background(), flights.FlightSearchRequest{
		Origin: "GRU", Destination: "JFK", Date: "2026-06-10",
	})

	if !errors.Is(err, flights.ErrAllProvidersFailed) {
		t.Errorf("expected ErrAllProvidersFailed, got %v", err)
	}

	var apfErr *flights.AllProvidersFailedError
	if !errors.As(err, &apfErr) {
		t.Fatalf("expected *AllProvidersFailedError, got %T", err)
	}
	if len(apfErr.ProviderErrors) != 1 {
		t.Errorf("expected 1 provider error from panic, got %d", len(apfErr.ProviderErrors))
	}
}

// --- Handler error handling tests ---

func TestHandler_Search_LowercaseIATANormalized(t *testing.T) {
	p := &errHandlingProvider{
		name:    "P1",
		results: []flights.Flight{makeErrHandlingFlight(2000, 500)},
	}
	h := flights.NewHandler(newErrHandlingSvc(p))

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=gru&destination=jfk&date=2026-06-10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — lowercase IATA should be normalized", rr.Code)
	}
}

func TestHandler_Search_502_IncludesProviderErrors(t *testing.T) {
	p := &errHandlingProvider{name: "P1", err: errors.New("upstream timeout")}
	h := flights.NewHandler(newErrHandlingSvc(p))

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rr.Code)
	}

	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Details []struct {
			Provider string `json:"provider"`
			Message  string `json:"message"`
		} `json:"details"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Error != "provider_unavailable" {
		t.Errorf("expected error=provider_unavailable, got %s", body.Error)
	}
	if len(body.Details) != 1 {
		t.Errorf("expected 1 provider error in details, got %d", len(body.Details))
	}
	if body.Details[0].Provider != "P1" {
		t.Errorf("expected provider P1, got %s", body.Details[0].Provider)
	}
}

func TestHandler_Search_ClientContextCancelled_NoResponse(t *testing.T) {
	// Provider that blocks until context is done.
	blocker := &slowErrProvider{delay: 200 * time.Millisecond}

	svc := flights.NewService([]flights.FlightProvider{blocker}, 5, 0)
	h := flights.NewHandler(svc)

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil).
		WithContext(ctx)
	rr := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.Search(rr, req)
	}()

	// Cancel the client context before the provider finishes.
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done

	// No status must have been written (WriteHeader not called) OR status is 0
	// (httptest.ResponseRecorder default). The handler must not panic.
	if rr.Code == http.StatusInternalServerError {
		t.Error("handler wrote 500 instead of silently dropping cancelled request")
	}
}

func TestHandler_Search_ValidationError_IncludesField(t *testing.T) {
	p := &errHandlingProvider{name: "P1"}
	h := flights.NewHandler(newErrHandlingSvc(p))

	req := httptest.NewRequest(http.MethodGet, "/flights/search?origin=GRU&destination=JFK&date=bad-format", nil)
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
	// Message must contain the field name.
	if body["message"] == "" {
		t.Error("expected non-empty message")
	}
}

// --- local stubs ---

func makeErrHandlingFlight(price float64, duration int) flights.Flight {
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

type slowErrProvider struct{ delay time.Duration }

func (s *slowErrProvider) Name() string { return "Slow" }
func (s *slowErrProvider) Search(ctx context.Context, _ flights.FlightSearchRequest) ([]flights.Flight, error) {
	select {
	case <-time.After(s.delay):
		return nil, errors.New("slow provider error")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sperax/flight-price-service/internal/config"
	"github.com/sperax/flight-price-service/internal/flights"
	"github.com/sperax/flight-price-service/internal/server"
)

// --- test infrastructure ---

// integrationProvider is a minimal FlightProvider for integration tests.
type integrationProvider struct {
	name string
}

func (p *integrationProvider) Name() string { return p.name }
func (p *integrationProvider) Search(_ context.Context, req flights.FlightSearchRequest) ([]flights.Flight, error) {
	now := time.Now()
	return []flights.Flight{
		{
			Provider:        p.name,
			Airline:         "TestAir",
			FlightNumber:    "TA001",
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   now,
			ArrivalTime:     now.Add(10 * time.Hour),
			DurationMinutes: 600,
			Price:           2500.00,
			Currency:        "BRL",
		},
		{
			Provider:        p.name,
			Airline:         "FastAir",
			FlightNumber:    "FA002",
			Origin:          req.Origin,
			Destination:     req.Destination,
			DepartureTime:   now,
			ArrivalTime:     now.Add(8 * time.Hour),
			DurationMinutes: 480,
			Price:           3500.00,
			Currency:        "BRL",
		},
	}, nil
}

// testConfig returns a deterministic config suitable for tests.
func testConfig() *config.Config {
	return &config.Config{
		AppPort:                "8080",
		AppEnv:                 "test",
		JWTSecret:              "integration-test-secret",
		JWTExpirationMinutes:   60,
		AuthUsername:           "testuser",
		AuthPassword:           "testpass",
		ProviderTimeoutSeconds: 3,
		CacheTTLSeconds:        30,
	}
}

// newTestServer builds the handler and wraps it in an httptest.Server.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := testConfig()
	providers := []flights.FlightProvider{&integrationProvider{name: "TestProvider"}}
	h := server.New(cfg, providers)
	return httptest.NewServer(h)
}

// login performs the login flow and returns a valid Bearer token.
func login(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": "testuser", "password": "testpass"})
	resp, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login returned %d, want 200", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}
	return payload.AccessToken
}

// --- health ---

func TestIntegration_Health(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body)
	}
}

// --- auth ---

func TestIntegration_Login_Success(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	token := login(t, srv)
	if token == "" {
		t.Error("expected non-empty access_token")
	}
}

func TestIntegration_Login_WrongPassword(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"username": "testuser", "password": "wrong"})
	resp, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestIntegration_Login_InvalidBody(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader([]byte("not-json")))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// --- flight search auth guard ---

func TestIntegration_FlightSearch_NoToken(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/flights/search?origin=GRU&destination=JFK&date=2026-06-10")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestIntegration_FlightSearch_InvalidToken(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// --- flight search happy path ---

func TestIntegration_FlightSearch_Success(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	token := login(t, srv)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var body flights.FlightSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.Origin != "GRU" {
		t.Errorf("expected origin GRU, got %s", body.Origin)
	}
	if body.Destination != "JFK" {
		t.Errorf("expected destination JFK, got %s", body.Destination)
	}
	if len(body.Flights) == 0 {
		t.Error("expected at least one flight")
	}
	if body.CheapestFlight == nil {
		t.Error("expected cheapest_flight to be set")
	}
	if body.FastestFlight == nil {
		t.Error("expected fastest_flight to be set")
	}
	if body.Cached {
		t.Error("expected cached=false on first request")
	}
}

func TestIntegration_FlightSearch_LowercaseIATA(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	token := login(t, srv)

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/flights/search?origin=gru&destination=jfk&date=2026-06-10", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for lowercase IATA, got %d", resp.StatusCode)
	}
}

func TestIntegration_FlightSearch_CacheHit(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	token := login(t, srv)

	doSearch := func() flights.FlightSearchResponse {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/flights/search?origin=GRU&destination=JFK&date=2026-06-10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()
		var body flights.FlightSearchResponse
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return body
	}

	first := doSearch()
	second := doSearch()

	if first.Cached {
		t.Error("expected first response cached=false")
	}
	if !second.Cached {
		t.Error("expected second response cached=true")
	}
}

// --- flight search validation ---

func TestIntegration_FlightSearch_MissingParams(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	token := login(t, srv)

	cases := []struct {
		url  string
		desc string
	}{
		{"/flights/search?destination=JFK&date=2026-06-10", "missing origin"},
		{"/flights/search?origin=GRU&date=2026-06-10", "missing destination"},
		{"/flights/search?origin=GRU&destination=JFK", "missing date"},
		{"/flights/search?origin=GRU&destination=JFK&date=10-06-2026", "invalid date format"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, srv.URL+tc.url, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("[%s] expected 400, got %d", tc.desc, resp.StatusCode)
			}

			var body map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&body)
			if body["error"] != "invalid_request" {
				t.Errorf("[%s] expected error=invalid_request, got %s", tc.desc, body["error"])
			}
		})
	}
}

// --- unknown routes ---

func TestIntegration_UnknownRoute(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/does-not-exist")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

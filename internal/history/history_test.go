package history_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sperax/flight-price-service/internal/history"
)

// --- handler tests ---

func TestHistory_MissingOrigin(t *testing.T) {
	h := history.NewHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flights/history?destination=JFK", nil)
	h.History(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %s", body["error"])
	}
}

func TestHistory_MissingDestination(t *testing.T) {
	h := history.NewHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flights/history?origin=GRU", nil)
	h.History(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %s", body["error"])
	}
}

func TestHistory_Success_Returns24Months(t *testing.T) {
	h := history.NewHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flights/history?origin=GRU&destination=JFK", nil)
	h.History(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp history.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Origin != "GRU" {
		t.Errorf("expected origin GRU, got %s", resp.Origin)
	}
	if resp.Destination != "JFK" {
		t.Errorf("expected destination JFK, got %s", resp.Destination)
	}
	if len(resp.History) != 24 {
		t.Errorf("expected 24 monthly entries, got %d", len(resp.History))
	}
}

func TestHistory_ResponseShape(t *testing.T) {
	h := history.NewHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flights/history?origin=GRU&destination=JFK", nil)
	h.History(w, r)

	var resp history.Response
	_ = json.NewDecoder(w.Body).Decode(&resp)

	now := time.Now()
	twoYearsAgo := now.AddDate(-2, 0, 0)

	for i, m := range resp.History {
		if m.Year == 0 {
			t.Errorf("entry %d: Year is zero", i)
		}
		if m.Month < 1 || m.Month > 12 {
			t.Errorf("entry %d: Month %d out of range [1,12]", i, m.Month)
		}
		if m.AvgPrice <= 0 {
			t.Errorf("entry %d: AvgPrice must be positive, got %f", i, m.AvgPrice)
		}
		if m.Currency != "BRL" {
			t.Errorf("entry %d: expected currency BRL, got %s", i, m.Currency)
		}
		entryTime := time.Date(m.Year, time.Month(m.Month), 1, 0, 0, 0, 0, time.UTC)
		if entryTime.Before(twoYearsAgo) {
			t.Errorf("entry %d: %d/%02d is older than 2 years", i, m.Year, m.Month)
		}
	}
}

func TestHistory_LowercaseNormalization(t *testing.T) {
	h := history.NewHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flights/history?origin=gru&destination=jfk", nil)
	h.History(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for lowercase IATA, got %d", w.Code)
	}
	var resp history.Response
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Origin != "GRU" {
		t.Errorf("expected normalized origin GRU, got %s", resp.Origin)
	}
	if resp.Destination != "JFK" {
		t.Errorf("expected normalized destination JFK, got %s", resp.Destination)
	}
}

func TestHistory_Deterministic(t *testing.T) {
	h := history.NewHandler()

	callHistory := func() history.Response {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/flights/history?origin=GRU&destination=JFK", nil)
		h.History(w, r)
		var resp history.Response
		_ = json.NewDecoder(w.Body).Decode(&resp)
		return resp
	}

	first := callHistory()
	second := callHistory()

	if len(first.History) != len(second.History) {
		t.Fatalf("history length differs between calls: %d vs %d", len(first.History), len(second.History))
	}
	for i := range first.History {
		if first.History[i].AvgPrice != second.History[i].AvgPrice {
			t.Errorf("entry %d: price differs between calls: %f vs %f",
				i, first.History[i].AvgPrice, second.History[i].AvgPrice)
		}
	}
}

func TestHistory_ContentTypeJSON(t *testing.T) {
	h := history.NewHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/flights/history?origin=GRU&destination=JFK", nil)
	h.History(w, r)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

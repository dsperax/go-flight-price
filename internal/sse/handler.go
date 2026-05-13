package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sperax/flight-price-service/internal/flights"
	"github.com/sperax/flight-price-service/internal/httpx"
)

// Searcher is the minimal interface the SSE handler requires from the flight service.
// Using an interface instead of the concrete *flights.Service keeps the handler testable.
type Searcher interface {
	Search(ctx context.Context, req flights.FlightSearchRequest) (flights.FlightSearchResponse, error)
}

// Handler handles Server-Sent Events subscription requests.
type Handler struct {
	svc          Searcher
	tickInterval time.Duration
}

// New creates a Handler with the production tick interval (30 s).
func New(svc Searcher) *Handler {
	return &Handler{
		svc:          svc,
		tickInterval: 30 * time.Second,
	}
}

// Subscribe handles GET /subscribe/{route}
//
// The {route} path parameter must be formatted as "ORIGIN-DESTINATION" (e.g. "GRU-JFK").
// An optional ?date=YYYY-MM-DD query param sets the search date; defaults to tomorrow.
//
// The endpoint streams SSE events: one immediately on connection, then every 30 seconds.
// Each event contains the full flight search response as a JSON payload.
func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	route := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "route")))
	parts := strings.SplitN(route, "-", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		httpx.BadRequest(w, "route must be formatted as ORIGIN-DESTINATION (e.g. GRU-JFK)")
		return
	}
	origin, destination := parts[0], parts[1]

	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.InternalError(w)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Emit an initial event so the client gets data immediately on connect.
	h.emit(w, flusher, origin, destination, date, r.Context())

	ticker := time.NewTicker(h.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.emit(w, flusher, origin, destination, date, r.Context())
		case <-r.Context().Done():
			return
		}
	}
}

// emit performs a flight search and writes the result as a single SSE data event.
// Errors are silently swallowed so the stream stays open across transient failures.
func (h *Handler) emit(w http.ResponseWriter, flusher http.Flusher, origin, destination, date string, ctx context.Context) {
	res, err := h.svc.Search(ctx, flights.FlightSearchRequest{
		Origin:      origin,
		Destination: destination,
		Date:        date,
	})
	if err != nil {
		return
	}
	data, err := json.Marshal(res)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

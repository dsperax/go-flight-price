package history

import (
	"net/http"
	"strings"

	"github.com/sperax/flight-price-service/internal/httpx"
)

// Handler holds the dependencies for the history HTTP endpoint.
type Handler struct{}

// NewHandler creates a new history Handler.
func NewHandler() *Handler { return &Handler{} }

// History handles GET /flights/history?origin=XXX&destination=YYY
// Returns 24 months of mock historical average prices for the given route.
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	origin := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("origin")))
	destination := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("destination")))

	if origin == "" {
		httpx.BadRequest(w, "origin is required")
		return
	}
	if destination == "" {
		httpx.BadRequest(w, "destination is required")
		return
	}

	resp := Response{
		Origin:      origin,
		Destination: destination,
		History:     GenerateHistory(origin, destination),
	}
	httpx.JSON(w, http.StatusOK, resp)
}

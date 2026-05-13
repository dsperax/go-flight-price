package flights

import (
	"errors"
	"net/http"

	"github.com/sperax/flight-price-service/internal/httpx"
)

// Handler holds the flight search HTTP handler dependencies.
type Handler struct {
	svc *Service
}

// NewHandler creates a new flight Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Search handles GET /flights/search?origin=&destination=&date=
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	req := FlightSearchRequest{
		Origin:      r.URL.Query().Get("origin"),
		Destination: r.URL.Query().Get("destination"),
		Date:        r.URL.Query().Get("date"),
	}

	if err := req.Validate(); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	resp, err := h.svc.Search(r.Context(), req)
	if err != nil {
		if errors.Is(err, ErrAllProvidersFailed) {
			httpx.Error(w, http.StatusBadGateway, "provider_unavailable", "all flight providers failed to return results")
			return
		}
		httpx.InternalError(w)
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

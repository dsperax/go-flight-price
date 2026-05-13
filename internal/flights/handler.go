package flights

import (
	"errors"
	"log"
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

	// Normalize before validation so users can pass lowercase IATA codes.
	req.Normalize()

	if err := req.Validate(); err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			httpx.BadRequest(w, valErr.Error())
			return
		}
		httpx.BadRequest(w, err.Error())
		return
	}

	resp, err := h.svc.Search(r.Context(), req)
	if err != nil {
		// If the HTTP request context is done the client already disconnected
		// (or a server-side deadline fired). Writing a response is pointless.
		if r.Context().Err() != nil {
			return
		}

		// All providers failed — surface each provider's individual error.
		var apfErr *AllProvidersFailedError
		if errors.As(err, &apfErr) {
			httpx.ErrorWithDetails(
				w,
				http.StatusBadGateway,
				"provider_unavailable",
				"all flight providers failed to return results",
				apfErr.ProviderErrors,
			)
			return
		}

		// Any other unexpected error — log it server-side and return a generic 500.
		log.Printf("ERROR [flights.Search] unexpected: %v", err)
		httpx.InternalError(w)
		return
	}

	httpx.JSON(w, http.StatusOK, resp)
}

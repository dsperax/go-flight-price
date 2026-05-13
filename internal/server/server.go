package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sperax/flight-price-service/internal/auth"
	"github.com/sperax/flight-price-service/internal/config"
	"github.com/sperax/flight-price-service/internal/flights"
	"github.com/sperax/flight-price-service/internal/history"
	"github.com/sperax/flight-price-service/internal/httpx"
	"github.com/sperax/flight-price-service/internal/sse"
)

// New builds and returns the application HTTP handler.
// Accepting providers as a parameter keeps the function testable without real API keys.
func New(cfg *config.Config, flightProviders []flights.FlightProvider) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	authHandler := auth.NewHandler(cfg.AuthUsername, cfg.AuthPassword, cfg.JWTSecret, cfg.JWTExpirationMinutes)
	r.Post("/auth/login", authHandler.Login)

	flightSvc := flights.NewService(flightProviders, cfg.ProviderTimeoutSeconds, cfg.CacheTTLSeconds)
	flightHandler := flights.NewHandler(flightSvc)
	histHandler := history.NewHandler()
	sseHandler := sse.New(flightSvc)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))
		r.Get("/flights/search", flightHandler.Search)
		r.Get("/flights/history", histHandler.History)
		r.Get("/subscribe/{route}", sseHandler.Subscribe)
	})

	return r
}

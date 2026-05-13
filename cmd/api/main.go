package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/sperax/flight-price-service/internal/auth"
	"github.com/sperax/flight-price-service/internal/config"
	"github.com/sperax/flight-price-service/internal/flights"
	"github.com/sperax/flight-price-service/internal/httpx"
	"github.com/sperax/flight-price-service/internal/providers"
)

func main() {
	// Load .env if present; ignore error in production where env vars are set directly.
	_ = godotenv.Load()

	cfg := config.Load()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	authHandler := auth.NewHandler(cfg.AuthUsername, cfg.AuthPassword, cfg.JWTSecret, cfg.JWTExpirationMinutes)
	r.Post("/auth/login", authHandler.Login)

	flightProviders := []flights.FlightProvider{
		providers.NewAmadeus(cfg.AmadeusAPIKey),
		providers.NewSkyscanner(cfg.SkyscannerAPIKey),
		providers.NewCheapFlights(cfg.CheapFlightsAPIKey),
	}
	flightSvc := flights.NewService(flightProviders, cfg.ProviderTimeoutSeconds)
	flightHandler := flights.NewHandler(flightSvc)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))
		r.Get("/flights/search", flightHandler.Search)
	})

	log.Printf("starting server env=%s port=%s", cfg.AppEnv, cfg.AppPort)
	if err := http.ListenAndServe(":"+cfg.AppPort, r); err != nil {
		log.Fatal(err)
	}
}

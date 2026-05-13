package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/sperax/flight-price-service/internal/config"
	"github.com/sperax/flight-price-service/internal/flights"
	"github.com/sperax/flight-price-service/internal/providers"
	"github.com/sperax/flight-price-service/internal/server"
)

func main() {
	// Load .env if present; ignore error in production where env vars are set directly.
	_ = godotenv.Load()

	cfg := config.Load()

	flightProviders := []flights.FlightProvider{
		providers.NewAmadeus(cfg.AmadeusAPIKey),
		providers.NewSkyscanner(cfg.SkyscannerAPIKey),
		providers.NewCheapFlights(cfg.CheapFlightsAPIKey),
	}

	h := server.New(cfg, flightProviders)

	log.Printf("starting server env=%s port=%s", cfg.AppEnv, cfg.AppPort)
	if err := http.ListenAndServe(":"+cfg.AppPort, h); err != nil {
		log.Fatal(err)
	}
}

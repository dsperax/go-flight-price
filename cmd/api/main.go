package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/sperax/flight-price-service/internal/config"
	"github.com/sperax/flight-price-service/internal/httpx"
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

	log.Printf("starting server env=%s port=%s", cfg.AppEnv, cfg.AppPort)
	if err := http.ListenAndServe(":"+cfg.AppPort, r); err != nil {
		log.Fatal(err)
	}
}

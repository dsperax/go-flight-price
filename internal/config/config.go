package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppPort              string
	AppEnv               string
	JWTSecret            string
	JWTExpirationMinutes int

	AuthUsername string
	AuthPassword string

	ProviderTimeoutSeconds int
	CacheTTLSeconds        int

	AmadeusAPIKey      string
	SkyscannerAPIKey   string
	CheapFlightsAPIKey string
}

func Load() *Config {
	return &Config{
		AppPort:              getEnv("APP_PORT", "8080"),
		AppEnv:               getEnv("APP_ENV", "development"),
		JWTSecret:            getEnv("JWT_SECRET", "changeme"),
		JWTExpirationMinutes: getEnvInt("JWT_EXPIRATION_MINUTES", 60),

		AuthUsername: getEnv("AUTH_USERNAME", "admin"),
		AuthPassword: getEnv("AUTH_PASSWORD", "admin123"),

		ProviderTimeoutSeconds: getEnvInt("PROVIDER_TIMEOUT_SECONDS", 3),
		CacheTTLSeconds:        getEnvInt("CACHE_TTL_SECONDS", 30),

		AmadeusAPIKey:      getEnv("AMADEUS_API_KEY", ""),
		SkyscannerAPIKey:   getEnv("SKYSCANNER_API_KEY", ""),
		CheapFlightsAPIKey: getEnv("CHEAPFLIGHTS_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

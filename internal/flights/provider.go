package flights

import "context"

// FlightProvider is the contract that every provider adapter must implement.
// To replace a mock with a real API, implement this interface and swap it in main.go.
type FlightProvider interface {
	Name() string
	Search(ctx context.Context, req FlightSearchRequest) ([]Flight, error)
}

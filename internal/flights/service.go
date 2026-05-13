package flights

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Service orchestrates concurrent flight searches across multiple providers.
type Service struct {
	providers       []FlightProvider
	providerTimeout time.Duration
}

// NewService creates a new Service with the given providers and per-provider timeout.
func NewService(providers []FlightProvider, providerTimeoutSeconds int) *Service {
	return &Service{
		providers:       providers,
		providerTimeout: time.Duration(providerTimeoutSeconds) * time.Second,
	}
}

// providerResult is the internal struct used to collect results from each goroutine.
type providerResult struct {
	flights []Flight
	err     error
	name    string
}

// Search calls all providers concurrently and aggregates their results.
// It returns ErrAllProvidersFailed only when every provider fails.
// Partial failures are included in FlightSearchResponse.ProviderErrors.
func (s *Service) Search(ctx context.Context, req FlightSearchRequest) (FlightSearchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.providerTimeout)
	defer cancel()

	resultCh := make(chan providerResult, len(s.providers))

	var wg sync.WaitGroup
	for _, p := range s.providers {
		wg.Add(1)
		go func(p FlightProvider) {
			defer wg.Done()
			flights, err := p.Search(ctx, req)
			resultCh <- providerResult{
				flights: flights,
				err:     err,
				name:    p.Name(),
			}
		}(p)
	}

	// Close the channel once all goroutines finish so the range below can exit.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var allFlights []Flight
	var providerErrors []ProviderError

	for r := range resultCh {
		if r.err != nil {
			providerErrors = append(providerErrors, ProviderError{
				Provider: r.name,
				Message:  r.err.Error(),
			})
			continue
		}
		allFlights = append(allFlights, r.flights...)
	}

	if len(allFlights) == 0 {
		return FlightSearchResponse{}, ErrAllProvidersFailed
	}

	// Sort by price ascending; tie-break by duration ascending.
	sort.Slice(allFlights, func(i, j int) bool {
		if allFlights[i].Price != allFlights[j].Price {
			return allFlights[i].Price < allFlights[j].Price
		}
		return allFlights[i].DurationMinutes < allFlights[j].DurationMinutes
	})

	// selectBest is called after sorting so pointers remain valid.
	cheapest, fastest := selectBest(allFlights)

	return FlightSearchResponse{
		Origin:         req.Origin,
		Destination:    req.Destination,
		Date:           req.Date,
		CheapestFlight: cheapest,
		FastestFlight:  fastest,
		Flights:        allFlights,
		ProviderErrors: providerErrors,
	}, nil
}

// selectBest returns the cheapest and fastest flights from the aggregated slice.
// Cheapest: lowest price (tie-break: lower duration). Fastest: lowest duration (tie-break: lower price).
func selectBest(flights []Flight) (cheapest *Flight, fastest *Flight) {
	for i := range flights {
		f := &flights[i]
		if cheapest == nil || f.Price < cheapest.Price || (f.Price == cheapest.Price && f.DurationMinutes < cheapest.DurationMinutes) {
			cheapest = f
		}
		if fastest == nil || f.DurationMinutes < fastest.DurationMinutes || (f.DurationMinutes == fastest.DurationMinutes && f.Price < fastest.Price) {
			fastest = f
		}
	}
	return cheapest, fastest
}

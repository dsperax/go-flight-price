package flights

import "errors"

// ErrAllProvidersFailed is the sentinel checked with errors.Is.
var ErrAllProvidersFailed = errors.New("all flight providers failed to return results")

// ValidationError is returned by FlightSearchRequest.Validate when a field is
// missing or malformed. It carries the Field name so callers can surface it.
type ValidationError struct {
	Field   string // e.g. "origin", "date"
	Message string // human-readable description
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}

// AllProvidersFailedError is returned by Service.Search when every provider
// fails. It satisfies errors.Is(err, ErrAllProvidersFailed) and carries each
// provider's individual failure so callers can surface the details.
type AllProvidersFailedError struct {
	ProviderErrors []ProviderError
}

func (e *AllProvidersFailedError) Error() string {
	return ErrAllProvidersFailed.Error()
}

// Is makes errors.Is(err, ErrAllProvidersFailed) match *AllProvidersFailedError.
func (e *AllProvidersFailedError) Is(target error) bool {
	return target == ErrAllProvidersFailed
}

package flights

import "errors"

// ErrAllProvidersFailed is returned when every provider in the list fails.
var ErrAllProvidersFailed = errors.New("all flight providers failed to return results")

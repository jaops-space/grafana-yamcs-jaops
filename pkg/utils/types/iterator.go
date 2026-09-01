package types

import (
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/core/http"
)

// FetchFunction represents a function that fetches data for pagination.
// It receives the per-request query parameters (pagination token merged
// with any caller-supplied initial query) and returns a result of type T, a
// continuation token, and any error encountered. Implementations must pass
// query straight through to a ...WithQuery HTTP method - they must never
// mutate shared HTTPManager.Query state directly.
type FetchFunction[T any] func(query map[string]string) (T, string, error)

// PaginatedRequestIterator handles paginated requests, managing the fetching of results
// and continuation tokens. It allows iterating through paginated data in a flexible way.
type PaginatedRequestIterator[T any] struct {
	initialQuery  map[string]string // Stores the initial query parameters.
	isInitialized bool              // Flag to check if the iteration has started.
	apiContext    *http.HTTPManager // Context used for the request.
	fetchData     FetchFunction[T]  // Function to fetch data.
	continuation  string            // Token to fetch the next set of results.
}

// NewPaginatedRequestIterator initializes a new PaginatedRequestIterator with a context and a fetch function.
func NewPaginatedRequestIterator[T any](manager *http.HTTPManager, fetch FetchFunction[T]) *PaginatedRequestIterator[T] {
	return &PaginatedRequestIterator[T]{
		initialQuery:  make(map[string]string),
		isInitialized: false,
		apiContext:    manager,
		fetchData:     fetch,
		continuation:  "",
	}
}

// SetQuery allows setting custom query parameters for the iterator.
func (iterator *PaginatedRequestIterator[T]) SetQuery(query map[string]string) {
	for key, value := range query {
		iterator.initialQuery[key] = value
	}
}

// Next fetches the next result from the iterator.
// It builds a per-call query from the initial query parameters and the
// continuation token, if present, and passes it directly to fetchData -
// it never mutates shared HTTPManager state.
func (iterator *PaginatedRequestIterator[T]) Next() (T, error) {
	query := make(map[string]string, len(iterator.initialQuery)+1)

	// Add the initial query parameters
	for key, value := range iterator.initialQuery {
		query[key] = value
	}

	// Set the continuation token if present
	if iterator.continuation != "" {
		query["next"] = iterator.continuation
	}

	// Fetch data and handle the continuation token
	result, token, err := iterator.fetchData(query)
	iterator.continuation = token
	iterator.isInitialized = true

	// Reset continuation if error occurred
	if err != nil {
		iterator.continuation = ""
	}

	return result, err
}

// HasNext returns whether there are more results to fetch.
// It checks if the iteration has started and whether the continuation token is set.
func (iterator *PaginatedRequestIterator[T]) HasNext() bool {
	return !iterator.isInitialized || iterator.continuation != ""
}

package rerank

import "errors"

// Sentinel error categories allow callers to classify rerank failures without
// parsing human-readable error strings. Wrap these with safe context using
// fmt.Errorf("...: %w", ErrInvalidRequest).
//
// Safety contract: errors derived from these sentinels must never include query
// text, document text, authorization headers, endpoint userinfo, provider
// response bodies, or any other protected value.
var (
	// ErrInvalidRequest indicates the caller-supplied Request is malformed
	// (empty query, empty/duplicate document IDs, invalid TopN, model conflict,
	// oversized encoded request).
	ErrInvalidRequest = errors.New("invalid rerank request")

	// ErrInvalidResponse indicates the provider returned a response that failed
	// validation (wrong cardinality, missing/duplicate/out-of-range index,
	// non-finite score, model mismatch, trailing JSON, oversized body).
	ErrInvalidResponse = errors.New("invalid rerank response")

	// ErrUnavailable indicates the provider could not be reached or returned a
	// transport-level failure (network error, timeout, non-2xx status).
	ErrUnavailable = errors.New("rerank provider unavailable")

	// ErrRequestTooLarge indicates the encoded request exceeded the configured
	// request byte limit before being sent.
	ErrRequestTooLarge = errors.New("rerank request too large")

	// ErrResponseTooLarge indicates the response body exceeded the configured
	// response byte limit before decoding completed.
	ErrResponseTooLarge = errors.New("rerank response too large")
)

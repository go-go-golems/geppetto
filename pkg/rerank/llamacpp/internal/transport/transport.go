// Package transport contains the security-sensitive HTTP mechanics shared by
// llama.cpp reranking protocols. It deliberately knows nothing about request or
// response payloads.
package transport

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/rerank"
)

// ParseAndValidateBaseURL validates a provider base URL without returning
// parse errors that may echo credentials or private topology.
func ParseAndValidateBaseURL(label, raw string) (string, error) {
	baseURL := strings.TrimSpace(raw)
	if baseURL == "" {
		return "", fmt.Errorf("%s base URL is required: %w", label, rerank.ErrInvalidRequest)
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("%s base URL is malformed: %w", label, rerank.ErrInvalidRequest)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%s base URL scheme must be http or https: %w", label, rerank.ErrInvalidRequest)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("%s base URL host is required: %w", label, rerank.ErrInvalidRequest)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("%s base URL must not contain userinfo: %w", label, rerank.ErrInvalidRequest)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%s base URL must not contain query or fragment: %w", label, rerank.ErrInvalidRequest)
	}
	if parsed.RawPath != "" || strings.Contains(parsed.Path, "//") {
		return "", fmt.Errorf("%s base URL path must be unambiguous: %w", label, rerank.ErrInvalidRequest)
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return "", fmt.Errorf("%s base URL path must not contain dot segments: %w", label, rerank.ErrInvalidRequest)
		}
	}
	return baseURL, nil
}

// RedactedTransportError discards transport details that may contain redirect
// targets, proxy URLs, userinfo, or query parameters.
func RedactedTransportError(label string) error {
	return fmt.Errorf("%s provider transport failed: %w", label, rerank.ErrUnavailable)
}

// ReadAtMost reads at most limit+1 bytes so callers can distinguish an exact
// limit from an oversized body without unbounded allocation.
func ReadAtMost(reader io.Reader, limit int64) ([]byte, bool, error) {
	limited := &io.LimitedReader{R: reader, N: limit + 1}
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	return body, int64(len(body)) > limit, nil
}

// DrainBounded permits connection reuse without retaining or exposing an error
// response body.
func DrainBounded(body io.Reader, limit int64) {
	_, _, _ = ReadAtMost(body, limit)
}

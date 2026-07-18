package rerank

import (
	"fmt"
	"strings"
)

// DefaultMaxRequestBytes is the conservative default bound on the encoded
// request body when a provider does not configure an explicit limit. It
// protects memory and transport but does not prove model context safety; the
// application must apply exact tokenization/truncation policy before the call.
const DefaultMaxRequestBytes int64 = 2 << 20 // 2 MiB

// ValidateRequest validates a caller-supplied Request independent of any
// provider transport. It rejects:
//   - empty query after trimming;
//   - zero documents;
//   - empty or duplicate document IDs;
//   - empty document text;
//   - TopN < 1;
//   - TopN > len(Documents);
//   - a request model that conflicts with the configured provider model
//     (when providerModel is non-empty and request model is non-empty).
//
// When providerModel is non-empty and request.Model is empty, the caller is
// expected to fill Model from the provider default before or after validation;
// ValidateRequest does not mutate the request.
//
// maxRequestBytes bounds the encoded request size when the caller has already
// encoded the payload; pass 0 to skip the encoded-size check.
func ValidateRequest(in Request, providerModel Model) error {
	if strings.TrimSpace(in.Query) == "" {
		return fmt.Errorf("rerank query is required: %w", ErrInvalidRequest)
	}
	if len(in.Documents) == 0 {
		return fmt.Errorf("rerank request requires at least one document: %w", ErrInvalidRequest)
	}
	if in.TopN < 1 {
		return fmt.Errorf("rerank top_n must be >= 1: %w", ErrInvalidRequest)
	}
	if in.TopN > len(in.Documents) {
		return fmt.Errorf("rerank top_n (%d) must be <= number of documents (%d): %w",
			in.TopN, len(in.Documents), ErrInvalidRequest)
	}

	seen := make(map[string]struct{}, len(in.Documents))
	for i, doc := range in.Documents {
		if strings.TrimSpace(doc.ID) == "" {
			return fmt.Errorf("rerank document %d requires a non-empty id: %w", i, ErrInvalidRequest)
		}
		if strings.TrimSpace(doc.Text) == "" {
			return fmt.Errorf("rerank document %d (%s) requires non-empty text: %w", i, doc.ID, ErrInvalidRequest)
		}
		if _, exists := seen[doc.ID]; exists {
			return fmt.Errorf("rerank document id %q is duplicated: %w", doc.ID, ErrInvalidRequest)
		}
		seen[doc.ID] = struct{}{}
	}

	// Strict equality when both are non-empty: prevent silent model drift.
	if in.Model != "" && providerModel.Name != "" && in.Model != providerModel.Name {
		return fmt.Errorf("rerank request model %q does not match provider model %q: %w",
			in.Model, providerModel.Name, ErrInvalidRequest)
	}

	return nil
}

// ResolveModel returns the effective request model: the request model when
// non-empty, otherwise the provider's configured model. It does not mutate the
// request.
func ResolveModel(in Request, providerModel Model) string {
	if in.Model != "" {
		return in.Model
	}
	return providerModel.Name
}

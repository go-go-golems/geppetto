// Package cohere implements a strict Cohere v2 /rerank adapter for the
// transport-neutral rerank.Provider interface.
//
// The adapter enforces:
//   - bounded request encoding (MaxRequestBytes) before sending;
//   - bounded response reading (MaxResponseBytes) before decoding;
//   - strict JSON decoding that rejects unknown fields and trailing data;
//   - outbound URL policy via security.ValidateOutboundURL (scheme, host,
//     userinfo, local-network opt-in; HTTPS and no local networks by default,
//     which is correct for the hosted api.cohere.com endpoint);
//   - redirect rejection (the canonical hosted endpoint does not redirect;
//     failing closed keeps credentials from crossing origins);
//   - context cancellation propagation;
//   - safe errors that never include query/document text, the API key,
//     endpoint userinfo, or provider response bodies.
//
// Caller document IDs never enter the provider payload; the adapter retains a
// local index-to-ID table and maps results back to caller identity.
//
// Usage note: Cohere bills rerank calls in search units
// (meta.billed_units.search_units), not tokens. The adapter therefore leaves
// Response.Usage nil (the provider did not report token usage) and computes
// Response.Cost only when Options.CostPerSearch is configured and the response
// carries billed units. See GEPPETTO-RERANKER-002 decision record DR-3.
package cohere

// Package decision provides a transport-neutral decision-model primitive.
//
// Decisions are the fourth Geppetto model-service primitive, alongside
// inference (pkg/inference/engine), embeddings (pkg/embeddings) and rerank
// (pkg/rerank). A decision model ("System One" model, such as Jev or
// LunaRoute's djev) does not generate text. It receives a state (text or
// JSON) and a set of typed questions about that state, and returns a
// probability distribution over each question's declared answer space:
//
//   - Noul: the probability that a proposition is true;
//   - Choice: one of 1–255 named options, with per-option probabilities;
//   - Score: a fractional level on an ordered rubric of 1–10 levels.
//
// The package knows nothing about retrieval, routing or thresholds. Those
// belong to the caller; Gate is a pure helper that applies thresholds the
// caller passes in.
//
// Dependency direction:
//
//	application -> geppetto/pkg/decision/factory -> pkg/decision/systemone -> pkg/decision
//
// Core invariants enforced by this package:
//
//   - Question ids are non-empty, at most 64 bytes, and use [A-Za-z0-9_.-],
//     so they are safe to log and to echo in errors.
//   - A question sets only the criteria fields that belong to its type.
//   - Choice options are an ordered slice. Decision models are sensitive to
//     option order, so adapters must send options in slice order.
//   - A response has exactly one answer per question, of the question's type,
//     with finite probabilities in [0, 1] over exactly the declared space.
//
// Safety contract: errors never include state text, question text,
// authorization headers, endpoint userinfo or raw provider bodies. The one
// exception is ProviderError.Message, the provider's short validation message
// for 400/404 responses, capped at 200 bytes.
package decision

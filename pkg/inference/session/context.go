package session

import "context"

type sessionMetaContextKey string

const (
	sessionIDContextKey   sessionMetaContextKey = "session_id"
	inferenceIDContextKey sessionMetaContextKey = "inference_id"
	runTagsContextKey     sessionMetaContextKey = "run_tags"
)

// WithSessionMeta stores session and inference identifiers in context so
// downstream middleware/tooling can correlate work for a single run.
func WithSessionMeta(ctx context.Context, sessionID, inferenceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if sessionID != "" {
		ctx = context.WithValue(ctx, sessionIDContextKey, sessionID)
	}
	if inferenceID != "" {
		ctx = context.WithValue(ctx, inferenceIDContextKey, inferenceID)
	}
	return ctx
}

// SessionIDFromContext returns the session identifier attached with
// WithSessionMeta, or "" when unavailable.
func SessionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	sessionID, _ := ctx.Value(sessionIDContextKey).(string)
	return sessionID
}

// InferenceIDFromContext returns the inference identifier attached with
// WithSessionMeta, or "" when unavailable.
func InferenceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	inferenceID, _ := ctx.Value(inferenceIDContextKey).(string)
	return inferenceID
}

// WithRunTags stores per-run tags in context for downstream callbacks.
func WithRunTags(ctx context.Context, tags map[string]any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(tags) == 0 {
		return ctx
	}
	cloned := make(map[string]any, len(tags))
	for k, v := range tags {
		cloned[k] = v
	}
	return context.WithValue(ctx, runTagsContextKey, cloned)
}

// RunTagsFromContext returns per-run tags attached with WithRunTags.
func RunTagsFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	tags, _ := ctx.Value(runTagsContextKey).(map[string]any)
	if len(tags) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(tags))
	for k, v := range tags {
		cloned[k] = v
	}
	return cloned
}

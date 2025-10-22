package engine

import (
    "context"
    "net/http"
)

// DebugTap receives low-level provider breadcrumbs for development/debugging.
// Engines should treat calls to this interface as best-effort and optional.
type DebugTap interface {
    OnHTTP(req *http.Request, body []byte)
    OnHTTPResponse(resp *http.Response, body []byte)
    OnSSE(event string, data []byte)
    OnProviderObject(name string, v any)
    // OnTurnBeforeConversion captures the turn state before it's converted to provider format
    OnTurnBeforeConversion(turnYAML []byte)
}

type debugTapKey struct{}

// WithDebugTap installs a DebugTap into the context. Engines can fetch it via DebugTapFrom.
func WithDebugTap(ctx context.Context, tap DebugTap) context.Context {
    return context.WithValue(ctx, debugTapKey{}, tap)
}

// DebugTapFrom retrieves a DebugTap from context when present.
func DebugTapFrom(ctx context.Context) (DebugTap, bool) {
    v := ctx.Value(debugTapKey{})
    if v == nil { return nil, false }
    t, ok := v.(DebugTap)
    return t, ok
}



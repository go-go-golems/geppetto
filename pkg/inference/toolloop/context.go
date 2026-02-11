package toolloop

import (
	"context"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

// SnapshotHook captures snapshots of a Turn at defined phases (pre/post inference, post tools).
type SnapshotHook func(ctx context.Context, t *turns.Turn, phase string)

type snapshotHookKey struct{}

// WithTurnSnapshotHook attaches a snapshot hook to the context.
func WithTurnSnapshotHook(ctx context.Context, hook SnapshotHook) context.Context {
	if hook == nil {
		return ctx
	}
	return context.WithValue(ctx, snapshotHookKey{}, hook)
}

// TurnSnapshotHookFromContext returns the snapshot hook attached to the context, if any.
func TurnSnapshotHookFromContext(ctx context.Context) (SnapshotHook, bool) {
	v := ctx.Value(snapshotHookKey{})
	if v == nil {
		return nil, false
	}
	h, ok := v.(SnapshotHook)
	return h, ok && h != nil
}

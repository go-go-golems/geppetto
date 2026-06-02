package geppetto

import "context"

func (m *moduleRuntime) runtimeContext() context.Context {
	if m == nil || m.runtimeLifetimeContext == nil {
		return context.Background()
	}
	return m.runtimeLifetimeContext
}

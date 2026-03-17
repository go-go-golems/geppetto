package sections

import (
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
)

// GatherFlagsFromProfileRegistry is retained as a no-op middleware.
// Profiles no longer contribute step-setting flags.
func GatherFlagsFromProfileRegistry(
	_ []string,
	_ string,
	options ...fields.ParseOption,
) sources.Middleware {
	_ = options
	return func(next sources.HandlerFunc) sources.HandlerFunc {
		return next
	}
}

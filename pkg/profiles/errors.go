package profiles

import (
	"errors"
	"fmt"
)

var (
	ErrRegistryNotFound = errors.New("registry not found")
	ErrProfileNotFound  = errors.New("profile not found")
	ErrVersionConflict  = errors.New("version conflict")
	ErrValidation       = errors.New("validation error")
	ErrReadOnlyStore    = errors.New("store is read-only")
)

// VersionConflictError reports optimistic-locking failures.
type VersionConflictError struct {
	Resource string
	Slug     string
	Expected uint64
	Actual   uint64
}

func (e *VersionConflictError) Error() string {
	if e == nil {
		return ErrVersionConflict.Error()
	}
	return fmt.Sprintf("%s %q version conflict: expected=%d actual=%d", e.Resource, e.Slug, e.Expected, e.Actual)
}

func (e *VersionConflictError) Is(target error) bool { return target == ErrVersionConflict }

// ValidationError reports invalid domain data.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ErrValidation.Error()
	}
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", ErrValidation, e.Reason)
	}
	return fmt.Sprintf("%s (%s): %s", ErrValidation, e.Field, e.Reason)
}

func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

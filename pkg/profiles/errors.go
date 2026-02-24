package profiles

import (
	"errors"
	"fmt"
)

var (
	ErrRegistryNotFound = errors.New("registry not found")
	ErrProfileNotFound  = errors.New("profile not found")
	ErrVersionConflict  = errors.New("version conflict")
	ErrPolicyViolation  = errors.New("policy violation")
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

// PolicyViolationError reports attempts to mutate/use profiles against policy.
type PolicyViolationError struct {
	ProfileSlug ProfileSlug
	Reason      string
}

func (e *PolicyViolationError) Error() string {
	if e == nil {
		return ErrPolicyViolation.Error()
	}
	if e.ProfileSlug.IsZero() {
		return fmt.Sprintf("%s: %s", ErrPolicyViolation, e.Reason)
	}
	return fmt.Sprintf("%s for profile %q: %s", ErrPolicyViolation, e.ProfileSlug, e.Reason)
}

func (e *PolicyViolationError) Is(target error) bool { return target == ErrPolicyViolation }

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

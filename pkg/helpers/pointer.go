package helpers

// Float64Pointer returns a pointer to the given float64 value.
// This is a replacement for the Weaviate helpers.Float64Pointer function.
func Float64Pointer(f float64) *float64 {
	return &f
}
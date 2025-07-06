package helpers

import (
	"testing"
)

func TestFloat64Pointer(t *testing.T) {
	// Test with a positive float64 value
	val := 3.14
	ptr := Float64Pointer(val)
	
	if ptr == nil {
		t.Errorf("Float64Pointer returned nil for value %f", val)
	}
	
	if *ptr != val {
		t.Errorf("Float64Pointer returned %f, expected %f", *ptr, val)
	}
	
	// Test with zero
	zero := 0.0
	ptrZero := Float64Pointer(zero)
	
	if ptrZero == nil {
		t.Errorf("Float64Pointer returned nil for zero value")
	}
	
	if *ptrZero != zero {
		t.Errorf("Float64Pointer returned %f, expected %f", *ptrZero, zero)
	}
	
	// Test with negative value
	negative := -42.5
	ptrNeg := Float64Pointer(negative)
	
	if ptrNeg == nil {
		t.Errorf("Float64Pointer returned nil for negative value %f", negative)
	}
	
	if *ptrNeg != negative {
		t.Errorf("Float64Pointer returned %f, expected %f", *ptrNeg, negative)
	}
}
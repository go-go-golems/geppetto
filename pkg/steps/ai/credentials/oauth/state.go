package oauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
)

// NewState creates an unpredictable OAuth state value. Hosts retain it only
// until the callback and must bind it to the exact pending login attempt.
func NewState() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", errors.New("generate OAuth state")
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

// ValidateState compares a callback state against the pending state without
// exposing either value in an error.
func ValidateState(expected, received string) error {
	if expected == "" || received == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(received)) != 1 {
		return errors.New("OAuth state validation failed")
	}
	return nil
}

package api

import "encoding/json"

// APIError is a non-success response returned by the Claude API.
//
// Callers can use errors.As to inspect the HTTP status and Anthropic error type
// without parsing the human-readable message.
type APIError struct {
	StatusCode int
	ErrorType  string
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func newAPIError(statusCode int, responseBody []byte) (*APIError, error) {
	var response ErrorResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, err
	}

	return &APIError{
		StatusCode: statusCode,
		ErrorType:  response.Error.Type,
		Message:    response.Error.Message,
	}, nil
}

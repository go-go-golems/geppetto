package openai_responses

import "github.com/go-go-golems/geppetto/pkg/steps/ai/connection"

const responsesOperation = "responses"

func Routes() connection.Routes {
	return connection.Routes{
		responsesOperation: connection.Relative("responses"),
		"input-tokens":     connection.Relative("responses/input_tokens"),
	}
}

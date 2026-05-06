package openai_responses

import "github.com/go-go-golems/geppetto/pkg/turns"

const openAIResponsesNamespaceKey = "openai_responses"

var (
	keyOpenAIResponsesResponseID  = turns.BlockMetaK[string](openAIResponsesNamespaceKey, "response_id", 1)
	keyOpenAIResponsesOutputIndex = turns.BlockMetaK[int](openAIResponsesNamespaceKey, "output_index", 1)
	keyOpenAIResponsesItemType    = turns.BlockMetaK[string](openAIResponsesNamespaceKey, "item_type", 1)
	keyOpenAIResponsesStatus      = turns.BlockMetaK[string](openAIResponsesNamespaceKey, "status", 1)
)

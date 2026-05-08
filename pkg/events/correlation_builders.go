package events

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	SegmentTypeText      = "text"
	SegmentTypeReasoning = "reasoning"
	SegmentTypeTool      = "tool"

	StreamKindContent   = "content"
	StreamKindReasoning = "reasoning"
	StreamKindToolCall  = "tool_call"
)

// BuildProviderCallCorrelation creates a provider-call identity that is stable
// before provider-native response IDs are known. The provider response ID is
// preserved separately in ResponseID when available.
func BuildProviderCallCorrelation(provider, inferenceID, runID string, providerCallIndex int, responseID string) Correlation {
	provider = strings.TrimSpace(provider)
	inferenceID = strings.TrimSpace(inferenceID)
	runID = strings.TrimSpace(runID)
	responseID = strings.TrimSpace(responseID)

	scopeID := runID
	if scopeID == "" {
		scopeID = inferenceID
	}
	corr := Correlation{
		Provider:          provider,
		InferenceID:       inferenceID,
		RunID:             runID,
		ResponseID:        responseID,
		ProviderCallIndex: int32(providerCallIndex),
	}
	if provider != "" && scopeID != "" {
		corr.ProviderCallID = fmt.Sprintf("%s:%s:provider-call:%d", provider, scopeID, providerCallIndex)
		corr.CorrelationKey = corr.ProviderCallID
	}
	return corr
}

// BuildSegmentCorrelation creates a segment identity nested under a provider
// call. The provider object ID is optional; when present it is used to make the
// correlation key provider-native without involving rendered chat message IDs.
func BuildSegmentCorrelation(parent Correlation, providerObjectID string, segmentIndex int, segmentType string) Correlation {
	providerObjectID = strings.TrimSpace(providerObjectID)
	segmentType = strings.TrimSpace(segmentType)

	corr := parent
	idx := int32(segmentIndex)
	corr.SegmentIndex = idx
	corr.SegmentType = segmentType
	corr.StreamKind = streamKindForSegmentType(segmentType)
	corr.ParentCorrelationKey = parent.CorrelationKey
	if providerObjectID != "" {
		corr.ItemID = providerObjectID
	}
	if parent.ProviderCallID != "" && segmentType != "" {
		objectPart := "segment"
		if providerObjectID != "" {
			objectPart = providerObjectID
		}
		corr.SegmentID = fmt.Sprintf("%s:%s:%d:%s", parent.ProviderCallID, objectPart, segmentIndex, segmentType)
		corr.CorrelationKey = corr.SegmentID
	}
	return corr
}

// BuildChatCompletionsCorrelation creates the normalized identity used for
// OpenAI-compatible streamed Chat Completions. Provider-native object IDs remain
// in ResponseID/ToolCallID; CorrelationKey is the stable cross-layer join key.
func BuildChatCompletionsCorrelation(provider, responseID string, choiceIndex *int, streamKind, toolCallID string, toolCallIndex *int) Correlation {
	provider = strings.TrimSpace(provider)
	responseID = strings.TrimSpace(responseID)
	streamKind = strings.TrimSpace(streamKind)
	toolCallID = strings.TrimSpace(toolCallID)

	corr := Correlation{
		Provider:       provider,
		ResponseID:     responseID,
		StreamKind:     streamKind,
		ToolCallID:     toolCallID,
		SegmentType:    segmentTypeForStreamKind(streamKind),
		CorrelationKey: chatCompletionsCorrelationKey(provider, responseID, choiceIndex, streamKind, toolCallID, toolCallIndex),
	}
	if choiceIndex != nil {
		v := int32(*choiceIndex)
		corr.ChoiceIndex = &v
	}
	if toolCallIndex != nil {
		v := int32(*toolCallIndex)
		corr.ToolCallIndex = &v
	}
	return corr
}

// BuildResponsesCorrelation creates the normalized identity for OpenAI
// Responses API stream items and summaries while preserving provider-native
// item IDs.
func BuildResponsesCorrelation(provider, responseID, itemID string, outputIndex, summaryIndex *int) Correlation {
	provider = strings.TrimSpace(provider)
	responseID = strings.TrimSpace(responseID)
	itemID = strings.TrimSpace(itemID)

	corr := Correlation{
		Provider:       provider,
		ResponseID:     responseID,
		ItemID:         itemID,
		CorrelationKey: responsesCorrelationKey(provider, responseID, itemID, outputIndex, summaryIndex),
	}
	if outputIndex != nil {
		v := int32(*outputIndex)
		corr.OutputIndex = &v
	}
	if summaryIndex != nil {
		v := int32(*summaryIndex)
		corr.SummaryIndex = &v
	}
	return corr
}

// BuildClaudeProviderCallCorrelation creates the provider-call envelope identity
// for Anthropic/Claude message streams.
func BuildClaudeProviderCallCorrelation(provider, responseID string, providerCallIndex int) Correlation {
	provider = strings.TrimSpace(provider)
	responseID = strings.TrimSpace(responseID)
	corr := Correlation{
		Provider:          provider,
		ResponseID:        responseID,
		ProviderCallIndex: int32(providerCallIndex),
	}
	if responseID != "" {
		corr.ProviderCallID = responseID
	} else if provider != "" {
		corr.ProviderCallID = fmt.Sprintf("%s-provider-call-%d", provider, providerCallIndex)
	}
	if provider != "" && corr.ProviderCallID != "" {
		corr.CorrelationKey = fmt.Sprintf("%s:%s:provider-call", provider, corr.ProviderCallID)
	}
	return corr
}

// BuildClaudeSegmentCorrelation creates content-block identity beneath a Claude
// provider-call envelope. ContentBlockIndex is provider-native and stable within
// a single message stream.
func BuildClaudeSegmentCorrelation(provider, providerCallID string, contentBlockIndex int, segmentType string) Correlation {
	provider = strings.TrimSpace(provider)
	providerCallID = strings.TrimSpace(providerCallID)
	segmentType = strings.TrimSpace(segmentType)
	idx := int32(contentBlockIndex)
	corr := Correlation{
		Provider:          provider,
		ProviderCallID:    providerCallID,
		ContentBlockIndex: &idx,
		SegmentIndex:      idx,
		SegmentType:       segmentType,
		StreamKind:        streamKindForSegmentType(segmentType),
	}
	if provider != "" && providerCallID != "" && segmentType != "" {
		corr.SegmentID = fmt.Sprintf("%s:block:%d:%s", providerCallID, contentBlockIndex, segmentType)
		corr.CorrelationKey = fmt.Sprintf("%s:%s:block:%d:%s", provider, providerCallID, contentBlockIndex, segmentType)
		corr.ParentCorrelationKey = fmt.Sprintf("%s:%s:provider-call", provider, providerCallID)
	}
	return corr
}

func chatCompletionsCorrelationKey(provider, responseID string, choiceIndex *int, streamKind, toolCallID string, toolCallIndex *int) string {
	if provider == "" || responseID == "" || streamKind == "" || streamKind == "unknown" {
		return ""
	}
	choice := 0
	if choiceIndex != nil {
		choice = *choiceIndex
	}
	if streamKind == StreamKindToolCall || streamKind == "tool_call" {
		if toolCallID != "" {
			return fmt.Sprintf("%s-chat:%s:choice:%d:tool:%s", provider, responseID, choice, toolCallID)
		}
		if toolCallIndex != nil {
			return fmt.Sprintf("%s-chat:%s:choice:%d:tool-index:%d", provider, responseID, choice, *toolCallIndex)
		}
	}
	return fmt.Sprintf("%s-chat:%s:choice:%d:%s", provider, responseID, choice, streamKind)
}

func responsesCorrelationKey(provider, responseID, itemID string, outputIndex, summaryIndex *int) string {
	if provider == "" || responseID == "" {
		return ""
	}
	if itemID != "" {
		return provider + ":" + responseID + ":item:" + itemID
	}
	if outputIndex != nil && summaryIndex != nil {
		return provider + ":" + responseID + ":output:" + strconv.Itoa(*outputIndex) + ":summary:" + strconv.Itoa(*summaryIndex)
	}
	if outputIndex != nil {
		return provider + ":" + responseID + ":output:" + strconv.Itoa(*outputIndex)
	}
	return provider + ":" + responseID
}

func segmentTypeForStreamKind(streamKind string) string {
	switch streamKind {
	case StreamKindContent:
		return SegmentTypeText
	case StreamKindReasoning:
		return SegmentTypeReasoning
	case StreamKindToolCall:
		return SegmentTypeTool
	default:
		return ""
	}
}

func streamKindForSegmentType(segmentType string) string {
	switch segmentType {
	case SegmentTypeText:
		return StreamKindContent
	case SegmentTypeReasoning:
		return StreamKindReasoning
	case SegmentTypeTool:
		return StreamKindToolCall
	default:
		return ""
	}
}

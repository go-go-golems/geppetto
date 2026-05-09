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

// BuildRunCorrelation creates the canonical run-level identity.
func BuildRunCorrelation(sessionID, runID, turnID string) Correlation {
	return Correlation{
		SessionID: strings.TrimSpace(sessionID),
		RunID:     strings.TrimSpace(runID),
		TurnID:    strings.TrimSpace(turnID),
	}
}

// BuildProviderCallCorrelation creates a provider-call identity that is stable
// before provider-native response IDs are known. The index is used only to build
// a deterministic ID; it is not exposed as canonical correlation state.
func BuildProviderCallCorrelation(provider, runID, _ string, providerCallIndex int, _ string) Correlation {
	provider = strings.TrimSpace(provider)
	runID = strings.TrimSpace(runID)
	corr := Correlation{RunID: runID}
	if provider != "" && runID != "" {
		corr.ProviderCallID = fmt.Sprintf("%s:%s:provider-call:%d", provider, runID, providerCallIndex)
	}
	return corr
}

// BuildSegmentCorrelation creates a segment identity nested under a provider
// call. The provider object ID is optional; when present it is used to make the
// segment ID provider-native without involving rendered chat message IDs.
func BuildSegmentCorrelation(parent Correlation, providerObjectID string, segmentOrdinal int, segmentType string) Correlation {
	providerObjectID = strings.TrimSpace(providerObjectID)
	segmentType = strings.TrimSpace(segmentType)

	corr := parent
	if parent.ProviderCallID != "" && segmentType != "" {
		objectPart := "segment"
		if providerObjectID != "" {
			objectPart = providerObjectID
		}
		corr.SegmentID = fmt.Sprintf("%s:%s:%d:%s", parent.ProviderCallID, objectPart, segmentOrdinal, segmentType)
	}
	return corr
}

// BuildToolCorrelation creates a canonical tool identity under a provider call.
func BuildToolCorrelation(parent Correlation, toolCallID string) Correlation {
	corr := parent
	corr.ToolCallID = strings.TrimSpace(toolCallID)
	return corr
}

// BuildChatCompletionsCorrelation creates the normalized segment identity used
// for OpenAI-compatible streamed Chat Completions.
func BuildChatCompletionsCorrelation(provider, responseID string, choiceIndex *int, streamKind, toolCallID string, toolCallIndex *int) Correlation {
	segmentID := ChatCompletionsSegmentID(provider, responseID, choiceIndex, streamKind, toolCallID, toolCallIndex)
	corr := Correlation{SegmentID: segmentID}
	if strings.TrimSpace(toolCallID) != "" {
		corr.ToolCallID = strings.TrimSpace(toolCallID)
	}
	return corr
}

// ChatCompletionsSegmentID returns the opaque canonical segment ID for an
// OpenAI-compatible Chat Completions stream.
func ChatCompletionsSegmentID(provider, responseID string, choiceIndex *int, streamKind, toolCallID string, toolCallIndex *int) string {
	provider = strings.TrimSpace(provider)
	responseID = strings.TrimSpace(responseID)
	streamKind = strings.TrimSpace(streamKind)
	toolCallID = strings.TrimSpace(toolCallID)
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

// BuildResponsesCorrelation creates the normalized segment identity for OpenAI
// Responses API stream items and summaries while preserving provider-native
// item IDs inside the opaque SegmentID.
func BuildResponsesCorrelation(provider, responseID, itemID string, outputIndex, summaryIndex *int) Correlation {
	return Correlation{SegmentID: ResponsesSegmentID(provider, responseID, itemID, outputIndex, summaryIndex)}
}

// ResponsesSegmentID returns the opaque canonical segment ID for an OpenAI
// Responses item/output/summary.
func ResponsesSegmentID(provider, responseID, itemID string, outputIndex, summaryIndex *int) string {
	provider = strings.TrimSpace(provider)
	responseID = strings.TrimSpace(responseID)
	itemID = strings.TrimSpace(itemID)
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

// BuildClaudeProviderCallCorrelation creates the provider-call envelope identity
// for Anthropic/Claude message streams.
func BuildClaudeProviderCallCorrelation(provider, responseID string, providerCallIndex int) Correlation {
	provider = strings.TrimSpace(provider)
	responseID = strings.TrimSpace(responseID)
	corr := Correlation{}
	if responseID != "" {
		corr.ProviderCallID = provider + ":" + responseID
	} else if provider != "" {
		corr.ProviderCallID = fmt.Sprintf("%s-provider-call-%d", provider, providerCallIndex)
	}
	return corr
}

// BuildClaudeSegmentCorrelation creates content-block identity beneath a Claude
// provider-call envelope. ContentBlockIndex remains internal to ID construction.
func BuildClaudeSegmentCorrelation(provider, providerCallID string, contentBlockIndex int, segmentType string) Correlation {
	provider = strings.TrimSpace(provider)
	providerCallID = strings.TrimSpace(providerCallID)
	segmentType = strings.TrimSpace(segmentType)
	corr := Correlation{ProviderCallID: providerCallID}
	if provider != "" && providerCallID != "" && segmentType != "" {
		corr.SegmentID = fmt.Sprintf("%s:%s:block:%d:%s", provider, providerCallID, contentBlockIndex, segmentType)
	}
	return corr
}

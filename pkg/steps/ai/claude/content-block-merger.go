package claude

import (
	"sort"

	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ContentBlockMerger manages the streaming response from Claude AI API for chat completion.
// It processes various event types to reconstruct the full message response, handling
// multiple content blocks, metadata updates, and error conditions.
//
// The merger accumulates content from text and tool use blocks, manages message metadata,
// and provides access to the reconstructed response and any errors encountered.
//
// Usage:
//  1. Create a new merger with NewContentBlockMerger()
//  2. For each streaming event, call Add() to process and update the internal state
//  3. Use Text() to get the current accumulated response text
//  4. Access the full response with Response() or any errors with Error()
//
// The merger handles parallel stream fragments, ensuring proper ordering and
// combination of content blocks in the final response.
type ContentBlockMerger struct {
	metadata      chat.EventMetadata
	stepMetadata  *steps.StepMetadata
	response      *api.MessageResponse
	error         *api.Error
	contentBlocks map[int]*api.ContentBlock
	inputTokens   int // Track input tokens from start event
}

func NewContentBlockMerger(metadata chat.EventMetadata, stepMetadata *steps.StepMetadata) *ContentBlockMerger {
	return &ContentBlockMerger{
		metadata:      metadata,
		stepMetadata:  stepMetadata,
		contentBlocks: make(map[int]*api.ContentBlock),
		inputTokens:   0,
	}
}

// Text returns the accumulated main response so far.
// In the claude case, this is the concatenated list of all the individual text blocks so far
func (cbm *ContentBlockMerger) Text() string {
	res := ""
	// Create a slice to store the keys
	keys := make([]int, 0, len(cbm.contentBlocks))

	// Collect all keys from the map
	for k := range cbm.contentBlocks {
		keys = append(keys, k)
	}

	// Sort the keys in ascending order
	sort.Ints(keys)

	// Iterate over the sorted keys
	for _, k := range keys {
		res += cbm.contentBlocks[k].Text
	}

	return res
}

func (cbm *ContentBlockMerger) Response() *api.MessageResponse {
	return cbm.response
}

func (cbm *ContentBlockMerger) Error() *api.Error {
	return cbm.error
}

const ModelMetadataSlug = "claude_model"
const StopReasonMetadataSlug = "claude_stop_reason"
const StopSequenceMetadataSlug = "claude_stop_sequence"

// TODO(manuel, 2024-06-07) Unify counting usage across steps and LLM calls so that we can use it for openai and other completion APIs as well.

const MessageIdMetadataSlug = "claude_message_id"
const RoleMetadataSlug = "claude_role"

// updateUsage updates the usage statistics and metadata from an event usage
func (cbm *ContentBlockMerger) updateUsage(event api.StreamingEvent) {
	cbm.metadata.LLMMessageMetadata.Usage = nil
	if event.Usage != nil {
		cbm.metadata.LLMMessageMetadata.Usage = &conversation.Usage{
			InputTokens:  event.Usage.InputTokens,
			OutputTokens: event.Usage.OutputTokens,
		}
	}

	if event.Message != nil {
		cbm.metadata.LLMMessageMetadata.Usage = &conversation.Usage{
			InputTokens:  event.Message.Usage.InputTokens,
			OutputTokens: event.Message.Usage.OutputTokens,
		}
	}
	if event.Usage != nil {
		cbm.metadata.LLMMessageMetadata.Usage = &conversation.Usage{
			InputTokens:  event.Usage.InputTokens,
			OutputTokens: event.Usage.OutputTokens,
		}
	}

	if cbm.metadata.Usage == nil {
		return
	}

	if cbm.metadata.Usage.InputTokens > 0 {
		cbm.inputTokens = cbm.metadata.Usage.InputTokens
	} else {
		cbm.metadata.Usage.InputTokens = cbm.inputTokens
	}
}

func (cbm *ContentBlockMerger) Add(event api.StreamingEvent) ([]chat.Event, error) {
	// NOTE(manuel, 2024-06-04) This is where to continue: implement the block merger for claude, maybe test it in the main.go,
	// then properly implement the step and try it out (maybe also in its own main.go, as an example of how to use steps on their own.

	log.Trace().Object("event", event).Msg("ContentBlockMerger.Add")

	switch event.Type {
	case api.PingType:
		return []chat.Event{}, nil

	case api.MessageStartType:
		if event.Message == nil {
			return nil, errors.New("MessageStartType event must have a message")
		}
		cbm.response = event.Message
		cbm.stepMetadata.Metadata[ModelMetadataSlug] = event.Message.Model
		cbm.stepMetadata.Metadata[MessageIdMetadataSlug] = event.Message.ID
		cbm.stepMetadata.Metadata[RoleMetadataSlug] = event.Message.Role

		// Update event metadata with common fields
		cbm.metadata.Engine = event.Message.Model
		cbm.updateUsage(event)

		return []chat.Event{chat.NewStartEvent(cbm.metadata, cbm.stepMetadata)}, nil

	case api.MessageDeltaType:
		if event.Delta == nil {
			return nil, errors.New("MessageDeltaType event must have a delta")
		}
		if event.Delta.StopReason != "" {
			cbm.stepMetadata.Metadata[StopReasonMetadataSlug] = event.Delta.StopReason
			cbm.metadata.StopReason = &event.Delta.StopReason
		}
		if event.Delta.StopSequence != "" {
			cbm.stepMetadata.Metadata[StopSequenceMetadataSlug] = event.Delta.StopSequence
		}

		cbm.updateUsage(event)

		return []chat.Event{chat.NewPartialCompletionEvent(cbm.metadata, cbm.stepMetadata, "", cbm.response.FullText())}, nil

	case api.MessageStopType:
		if cbm.response == nil {
			return nil, errors.New("MessageStopType event must have a message to store the finished content block")
		}

		if event.Message != nil {
			if event.Message.StopReason != "" {
				cbm.stepMetadata.Metadata[StopReasonMetadataSlug] = event.Message.StopReason
				cbm.metadata.StopReason = &event.Message.StopReason
			}
			if event.Message.StopSequence != "" {
				cbm.stepMetadata.Metadata[StopSequenceMetadataSlug] = event.Message.StopSequence
			}
		}

		return []chat.Event{chat.NewFinalEvent(cbm.metadata, cbm.stepMetadata, cbm.response.FullText())}, nil

	case api.ContentBlockStartType:
		if cbm.response == nil {
			return nil, errors.New("ContentBlockStartType event must have a message to store the finished content block")
		}
		if event.ContentBlock == nil {
			return nil, errors.New("ContentBlockStartType event must have a content block")
		}
		if event.Index < 0 {
			return nil, errors.New("ContentBlockStartType event must have a positive index")
		}
		if _, exists := cbm.contentBlocks[event.Index]; exists {
			return nil, errors.Errorf("ContentBlockStartType event with index %d already exists", event.Index)
		}
		cbm.contentBlocks[event.Index] = event.ContentBlock

		// TODO(manuel, 2024-07-04) We should have a proper BlockStart message here
		return []chat.Event{}, nil

	case api.ContentBlockDeltaType:
		if cbm.response == nil {
			return nil, errors.New("ContentBlockDeltaType event must have a message to store the finished content block")
		}
		if event.Delta == nil {
			return nil, errors.New("ContentBlockDeltaType event must have a delta")
		}
		cb, exists := cbm.contentBlocks[event.Index]
		if !exists {
			return nil, errors.Errorf("ContentBlockDeltaType event with index %d does not exist", event.Index)
		}

		cbm.updateUsage(event)

		delta := ""
		switch event.Delta.Type {
		case api.TextDeltaType:
			delta = event.Delta.Text
			cb.Text += event.Delta.Text
			return []chat.Event{chat.NewPartialCompletionEvent(cbm.metadata, cbm.stepMetadata, delta, cbm.response.FullText()+cb.Text)}, nil
		case api.InputJSONDeltaType:
			delta = event.Delta.PartialJSON
			cb.Input += event.Delta.PartialJSON
			// TODO(manuel, 2024-07-04) This is where we would do partial tool call streaming
			_ = delta
		}
		return []chat.Event{}, nil

	case api.ContentBlockStopType:
		if cbm.response == nil {
			return nil, errors.New("ContentBlockStopType event must have a message to store the finished content block")
		}
		cb, exists := cbm.contentBlocks[event.Index]
		if !exists {
			return nil, errors.Errorf("ContentBlockStopType event with index %d does not exist", event.Index)
		}
		switch cb.Type {
		case api.ContentTypeText:
			cbm.response.Content = append(cbm.response.Content, api.NewTextContent(cb.Text))
			// TODO(manuel, 2024-07-04) This shoudl be some sort of block stop type
			return []chat.Event{chat.NewPartialCompletionEvent(cbm.metadata, cbm.stepMetadata, "", cbm.response.FullText())}, nil

		case api.ContentTypeToolUse:
			cbm.response.Content = append(cbm.response.Content, api.NewToolUseContent(cb.ID, cb.Name, cb.Input))
			return []chat.Event{chat.NewToolCallEvent(cbm.metadata, cbm.stepMetadata, chat.ToolCall{
				ID:    cb.ID,
				Name:  cb.Name,
				Input: cb.Input,
			})}, nil

		case api.ContentTypeImage, api.ContentTypeToolResult:
			return nil, errors.Errorf("Unsupported content block type: %s", cb.Type)
		}

		return nil, errors.Errorf("Unknown content block type: %s", cb.Type)

	case api.ErrorType:
		if event.Error == nil {
			return nil, errors.New("ErrorType event must have an error")
		}
		cbm.error = event.Error
		return []chat.Event{chat.NewErrorEvent(cbm.metadata, cbm.stepMetadata, event.Error.Message)}, nil

	default:
		return nil, errors.Errorf("Unknown event type: %s", event.Type)
	}
}

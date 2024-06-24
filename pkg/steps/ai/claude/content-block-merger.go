package claude

import (
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/claude/api"
	"github.com/pkg/errors"
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
}

func NewContentBlockMerger(metadata chat.EventMetadata, stepMetadata *steps.StepMetadata) *ContentBlockMerger {
	return &ContentBlockMerger{
		metadata:      metadata,
		stepMetadata:  stepMetadata,
		contentBlocks: make(map[int]*api.ContentBlock),
	}
}

// Text returns the accumulated main response so far. In the claude case, this is the concatenated list of all the individual text blocks so far
func (cbm *ContentBlockMerger) Text() string {
	panic("Not implemented!")
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

const UsageMetadataSlug = "claude_usage"
const MessageIdMetadataSlug = "claude_message_id"
const RoleMetadataSlug = "claude_role"

func (cbm *ContentBlockMerger) Add(event api.StreamingEvent) (*chat.EventPartialCompletion, error) {
	// NOTE(manuel, 2024-06-04) This is where to continue: implement the block merger for claude, maybe test it in the main.go,
	// then properly implement the step and try it out (maybe also in its own main.go, as an example of how to use steps on their own.

	switch event.Type {
	case api.PingType:
		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypePartial,
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
			Delta:      "",
			Completion: cbm.response.FullText(),
		}, nil

	case api.MessageStartType:
		if event.Message == nil {
			return nil, errors.New("MessageStartType event must have a message")
		}
		cbm.response = event.Message
		// TODO(manuel, 2024-06-05) Where do we store all the metadata we get from the model? in step metadata?
		cbm.stepMetadata.Metadata[ModelMetadataSlug] = event.Message.Model
		cbm.stepMetadata.Metadata[UsageMetadataSlug] = event.Message.Usage
		cbm.stepMetadata.Metadata[MessageIdMetadataSlug] = event.Message.ID
		cbm.stepMetadata.Metadata[RoleMetadataSlug] = event.Message.Role

		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypePartial,
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
			Delta:      "",
			Completion: cbm.response.FullText(),
		}, nil

	case api.MessageDeltaType:
		if event.Delta == nil {
			return nil, errors.New("MessageDeltaType event must have a delta")
		}
		// NOTE(manuel, 2024-06-07) I don't know if this means we need to append to the stop reason so far
		if event.Delta.StopReason != "" {
			cbm.stepMetadata.Metadata[StopReasonMetadataSlug] = event.Delta.StopReason
		}
		if event.Delta.StopSequence != "" {
			cbm.stepMetadata.Metadata[StopSequenceMetadataSlug] = event.Delta.StopSequence
		}
		// update the usage
		if event.Usage != nil {
			// Similarly, I don't know if we need to add to the usage count or if this represents the total so far.
			cbm.stepMetadata.Metadata[UsageMetadataSlug] = event.Usage
		}

		// create an "empty" partial completion event
		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypePartial,
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
			Delta:      "",
			Completion: cbm.response.FullText(),
		}, nil

	case api.MessageStopType:
		// for now, we concatenate all the content blocks, although it's unclear what to do if we have
		// both text and tool calls for example, and we might need to adjust the API to match
		// doing the same for openai as well.
		if cbm.response == nil {
			return nil, errors.New("MessageStopType event must have a message to store the finished content block")
		}

		if event.Message == nil {
			return nil, errors.New("MessageStopType event must have a message")

		}

		if event.Message.StopReason != "" {
			cbm.stepMetadata.Metadata[StopReasonMetadataSlug] = event.Message.StopReason
		}
		if event.Message.StopSequence != "" {
			cbm.stepMetadata.Metadata[StopSequenceMetadataSlug] = event.Message.StopSequence
		}

		cbm.stepMetadata.Metadata[UsageMetadataSlug] = event.Message.Usage

		// TODO(manuel, 2024-06-23) Hmm, shouldn't we send a completion event back here? Gosh I wish I had finished this project back then.
		//  Oh I remember the issue here, it's that there can be many messages within the same stream. No actually this should be th efinal message in the stream, iirc.
		//
		// how do we signal completion of the message here, since we also have to deal with signaling things from the event channel.
		// here we actually try to press too much into our current partial completion event, since claude has its own wrapping of streaming
		// a message status. For now let's return an empty result
		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypeFinal,
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
			// NOTE(manuel, 2024-06-05) This is not truly the case if we chain multiple blocks (for example, tool call following a string)
			// NOTE(manuel, 2024-06-23) I think this refers to returning a partial completion event with tools.
			Delta:      "",
			Completion: cbm.response.FullText(),
		}, nil

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

		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypePartial,
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
			// NOTE(manuel, 2024-06-05) This is not truly the case if we chain multiple blocks (for example, tool call following a string)
			Delta:      "",
			Completion: cbm.response.FullText(),
		}, nil

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
		delta := ""
		switch event.Delta.Type {
		case api.TextDeltaType:
			delta = event.Delta.Text
			cb.Text += event.Delta.Text
		case api.InputJSONDeltaType:
			delta = event.Delta.PartialJSON
			cb.Text += event.Delta.PartialJSON
		}

		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypePartial,
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
			Delta: delta,
			// prepend previously received content blocks
			Completion: cbm.response.FullText() + cb.Text,
		}, nil

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

		case api.ContentTypeToolUse:
			cbm.response.Content = append(cbm.response.Content, api.NewToolUseContent(cb.ID, cb.Name, cb.Input))
			// TODO(manuel, 2024-06-23) At this point we should take the tool call and put it in the metadata.

		case api.ContentTypeImage:
			// TODO(manuel, 2024-06-24) Handle encoded image data
			// cbm.response.Content = append(cbm.response.Content, api.NewImageContent(cb.Text))

		default:
			return nil, errors.Errorf("Unknown content block type: %s", cb.Type)
		}

		// not a real partial completion, we have to wait for message_stop
		// TODO(manuel, 2024-06-23) We still need to send back proper metadata and all that
		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypePartial,
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
			Delta:      "",
			Completion: cbm.response.FullText(),
		}, nil

	case api.ErrorType:
		if event.Error == nil {
			return nil, errors.New("ErrorType event must have an error")
		}
		cbm.error = event.Error
		return &chat.EventPartialCompletion{
			Event: chat.Event{
				Type:     chat.EventTypeError,
				Error:    errors.New(event.Error.Message),
				Metadata: cbm.metadata,
				Step:     cbm.stepMetadata,
			},
		}, nil

	default:
		return nil, errors.Errorf("Unknown event type: %s", event.Type)
	}
}

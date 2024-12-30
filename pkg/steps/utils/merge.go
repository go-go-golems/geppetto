package utils

import (
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

func NewMergeStep(manager conversation.Manager, prepend bool) steps.Step[*conversation.Message, conversation.Conversation] {
	// TODO(manuel, 2024-01-13) This should actually create an entirely new "conversation"
	// with the messages parentIds changed up. But this wouldn't be a manager nor maybe a single conversation?
	// Or maybe this just stays a conversation, linearly, with parentIds of the heads changed up
	//
	// This is currently only used in the codegen test anyway, so maybe this is a step that is not even going to be really used.
	mergeStep := &LambdaStep[*conversation.Message, conversation.Conversation]{
		Function: func(input *conversation.Message) helpers.Result[conversation.Conversation] {
			if prepend {
				manager.(*conversation.ManagerImpl).PrependMessages(input)
			} else {
				manager.(*conversation.ManagerImpl).AppendMessages(input)
			}

			return helpers.NewValueResult[conversation.Conversation](manager.GetConversation())
		},
	}
	return mergeStep
}

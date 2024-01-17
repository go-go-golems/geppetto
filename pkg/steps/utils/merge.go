package utils

import (
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

func NewMergeStep(manager conversation.Manager, prepend bool) steps.Step[string, conversation.Conversation] {
	// TODO(manuel, 2024-01-13) This should actually create an entirely new "conversation"
	// with the messages parentIds changed up. But this wouldn't be a manager nor maybe a single conversation?
	// Or maybe this just stays a conversation, linearly, with parentIds of the heads changed up
	//
	// This is currently only used in the codegen test anyway, so maybe this is a step that is not even going to be really used.
	mergeStep := &LambdaStep[string, conversation.Conversation]{
		Function: func(input string) helpers.Result[conversation.Conversation] {
			if prepend {
				// TODO(manuel, 2024-01-13) Hack for now because I'm not out to refactor the conversation manager interface yet
				// FIXME
				manager.(*conversation.ManagerImpl).PrependMessages(conversation.NewChatMessage(conversation.RoleUser, input))
			}

			manager.AppendMessages(conversation.NewChatMessage(conversation.RoleAssistant, input))
			return helpers.NewValueResult[conversation.Conversation](manager.GetConversation())
		},
	}
	return mergeStep
}

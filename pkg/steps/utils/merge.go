package utils

import (
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
)

func NewMergeStep(manager *context.Manager, prepend bool) steps.Step[string, []*conversation.Message] {
	mergeStep := &LambdaStep[string, []*conversation.Message]{
		Function: func(input string) helpers.Result[[]*conversation.Message] {
			if prepend {
				manager.PrependMessages(conversation.NewMessage(input, conversation.RoleUser))
			}

			manager.AddMessages(conversation.NewMessage(input, conversation.RoleAssistant))
			return helpers.NewValueResult[[]*conversation.Message](manager.GetMessages())
		},
	}
	return mergeStep
}

package utils

import (
	"github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"time"
)

func NewMergeStep(manager *context.Manager, prepend bool) steps.Step[string, []*context.Message] {
	mergeStep := &LambdaStep[string, []*context.Message]{
		Function: func(input string) helpers.Result[[]*context.Message] {
			if prepend {
				manager.PrependMessages(&context.Message{
					Text: input,
					Time: time.Now(),
					Role: context.RoleUser,
				})
			}

			manager.AddMessages(&context.Message{
				Text: input,
				Time: time.Now(),
				Role: context.RoleAssistant,
			})
			return helpers.NewValueResult[[]*context.Message](manager.GetMessagesWithSystemPrompt())
		},
	}
	return mergeStep
}

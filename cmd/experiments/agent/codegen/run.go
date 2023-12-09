package codegen

import (
	"context"
	"github.com/go-go-golems/geppetto/cmd/experiments/agent/helpers"
	context2 "github.com/go-go-golems/geppetto/pkg/context"
	helpers2 "github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openai2 "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/spf13/cobra"
	"time"
)

var CodegenTestCmd = &cobra.Command{
	Use:   "codegen-test",
	Short: "Test codegen prompt",
	Run: func(cmd *cobra.Command, args []string) {
		stepSettings, err := createSettingsFromCobra(cmd)
		cobra.CheckErr(err)

		stepFactory := &chat.StandardStepFactory{
			Settings: stepSettings,
		}

		c, err := NewTestCodegenCommand()
		cobra.CheckErr(err)

		c.StepFactory = stepFactory

		params := &TestCodegenCommandParameters{
			Pretend: "Scientist",
			What:    "Size of the moon",
			Of:      "My heart",
			Query:   []string{"What is the size of the moon?"},
		}

		ctx := context.Background()
		err = c.RunIntoWriter(ctx, params, cmd.OutOrStdout())
		cobra.CheckErr(err)
	},
}

func createSettingsFromCobra(cmd *cobra.Command) (*settings.StepSettings, error) {
	layer, err := openai2.NewParameterLayer()
	if err != nil {
		return nil, err
	}
	aiLayer, err := settings.NewChatParameterLayer()
	if err != nil {
		return nil, err
	}

	// TODO(manuel, 2023-11-28) Turn this into a "add all flags to command"
	// function to create commands, like glazedParameterLayer
	parsedLayers, err := helpers.ParseLayersFromCobraCommand(cmd, []cli.CobraParameterLayer{layer, aiLayer})
	if err != nil {
		return nil, err
	}

	stepSettings := settings.NewStepSettings()
	err = stepSettings.UpdateFromParsedLayers(parsedLayers)
	if err != nil {
		return nil, err
	}

	stepSettings.Chat.Stream = true

	return stepSettings, nil
}

var MultiStepCodgenTestCmd = &cobra.Command{
	Use:   "multi-step",
	Short: "Test codegen prompt",
	Run: func(cmd *cobra.Command, args []string) {
		stepSettings, err := createSettingsFromCobra(cmd)
		cobra.CheckErr(err)

		stepFactory := &chat.StandardStepFactory{
			Settings: stepSettings,
		}

		scientistCommand, err := NewTestCodegenCommand()
		cobra.CheckErr(err)
		scientistCommand.StepFactory = stepFactory

		scientistParams := &TestCodegenCommandParameters{
			Pretend: "Scientist",
			What:    "Size of the moon",
			Of:      "My heart",
			Query:   []string{"What is the size of the moon?"},
		}

		manager, err := scientistCommand.CreateManager(scientistParams)
		cobra.CheckErr(err)

		scientistStep, err := scientistCommand.CreateStep()

		writerParams := &TestCodegenCommandParameters{
			Pretend: "Writer",
			What:    "Biography of the scientist",
			Of:      "The previous story",
			Query:   []string{"Write a beautiful biography."},
		}
		writerCommand, err := NewTestCodegenCommand()
		cobra.CheckErr(err)
		writerCommand.StepFactory = stepFactory

		writerStep, err := writerCommand.CreateStep()

		var mergeStep steps.Step[string, []*context2.Message]
		mergeStep = &utils.LambdaStep[string, []*context2.Message]{
			Function: func(input string) helpers2.Result[[]*context2.Message] {
				messages := append(writerCommand.Messages, &context2.Message{
					Text: input,
					Time: time.Now(),
					Role: context2.RoleAssistant,
				})
				writerManager, err := context2.CreateManager(
					writerCommand.SystemPrompt, writerCommand.Prompt, messages, writerParams)
				if err != nil {
					return helpers2.NewErrorResult[[]*context2.Message](err)
				}
				return helpers2.NewValueResult[[]*context2.Message](writerManager.GetMessagesWithSystemPrompt())
			},
		}

		ctx := context.Background()
		var sRes steps.StepResult[string]
		sRes, err = scientistStep.Start(ctx, manager.GetMessagesWithSystemPrompt())
		cobra.CheckErr(err)

		sMerged := steps.Bind[string, []*context2.Message](ctx, sRes, mergeStep)
		s2 := steps.Bind[[]*context2.Message, string](ctx, sMerged, writerStep)

		res := s2.Return()
		for _, m := range res {
			_, err = cmd.OutOrStdout().Write([]byte(m.ValueOr("\nerror...\n")))
			cobra.CheckErr(err)
		}
	},
}

package codegen

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/go-go-golems/geppetto/cmd/experiments/agent/helpers"
	context2 "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openai2 "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"io"
)

var CodegenTestCmd = &cobra.Command{
	Use:   "codegen-test",
	Short: "Test codegen prompt",
	Run: func(cmd *cobra.Command, args []string) {
		stepSettings, err := createSettingsFromCobra(cmd)
		cobra.CheckErr(err)

		c, err := NewTestCodegenCommand()
		cobra.CheckErr(err)

		c.StepSettings = stepSettings

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

func printToStdout(s string, w io.Writer) error {
	_, err := w.Write([]byte(s))
	return err
}

type printer struct {
	Name    string
	w       io.Writer
	isFirst bool
}

func (p *printer) Print(s string) error {
	if p.isFirst {
		p.isFirst = false
		err := printToStdout(fmt.Sprintf("\n%s: \n", p.Name), p.w)
		if err != nil {
			return err
		}
	}
	return printToStdout(s, p.w)
}

func NewPrinterFunc(name string, w io.Writer) func(string) error {
	p := &printer{
		Name:    name,
		w:       w,
		isFirst: true,
	}
	return p.Print
}

func stepPrinterFunc(name string, w io.Writer) func(msg *message.Message) error {
	isFirst := true
	return func(msg *message.Message) error {
		e := &chat.Event{}
		err := json.Unmarshal(msg.Payload, e)
		if err != nil {
			return err
		}

		switch e.Type {
		case chat.EventTypeError:
			return err
		case chat.EventTypePartial:
			if isFirst {
				isFirst = false
				err := printToStdout(fmt.Sprintf("\n%s: \n", name), w)
				if err != nil {
					return err
				}
			}
			err := printToStdout(e.Text, w)
			if err != nil {
				return err
			}
		case chat.EventTypeFinal:
		case chat.EventTypeInterrupt:
		}

		msg.Ack()

		return nil
	}
}

var MultiStepCodgenTestCmd = &cobra.Command{
	Use:   "multi-step",
	Short: "Test codegen prompt",
	Run: func(cmd *cobra.Command, args []string) {
		stepSettings, err := createSettingsFromCobra(cmd)
		cobra.CheckErr(err)

		scientistCommand, err := NewTestCodegenCommand()
		cobra.CheckErr(err)
		scientistCommand.StepSettings = stepSettings

		scientistParams := &TestCodegenCommandParameters{
			Pretend: "Scientist",
			What:    "Size of the moon",
			Of:      "My heart",
			Query:   []string{"What is the size of the moon?"},
		}

		manager, err := scientistCommand.CreateManager(scientistParams)
		cobra.CheckErr(err)

		logger := watermill.NewStdLogger(false, false)
		pubSub := gochannel.NewGoChannel(gochannel.Config{
			// Guarantee that messages are delivered in the order of publishing.
			BlockPublishUntilSubscriberAck: true,
		}, logger)

		router, err := message.NewRouter(message.RouterConfig{}, logger)
		cobra.CheckErr(err)
		defer func(router *message.Router) {
			err := router.Close()
			if err != nil {
				log.Error().Err(err).Msg("Failed to close router")
			}
		}(router)

		router.AddNoPublisherHandler("scientist",
			"scientist",
			pubSub,
			stepPrinterFunc("Scientist", cmd.OutOrStdout()),
		)
		router.AddNoPublisherHandler("writer",
			"writer",
			pubSub,
			stepPrinterFunc("Writer", cmd.OutOrStdout()),
		)

		writerParams := &TestCodegenCommandParameters{
			Pretend: "Writer",
			What:    "Biography of the scientist",
			Of:      "The previous story",
			Query:   []string{"Write a beautiful biography."},
		}
		writerCommand, err := NewTestCodegenCommand()
		cobra.CheckErr(err)
		writerCommand.StepSettings = stepSettings

		writerManager, err := writerCommand.CreateManager(writerParams)
		cobra.CheckErr(err)

		scientistStep, err := scientistCommand.CreateStep(
			chat.WithSubscription(pubSub, "scientist"),
		)
		writerStep, err := writerCommand.CreateStep(
			chat.WithSubscription(pubSub, "writer"),
		)

		mergeStep := utils.NewMergeStep(writerManager, true)

		ctx := cmd.Context()

		errgrp := errgroup.Group{}
		errgrp.Go(func() error {
			var scientistResult steps.StepResult[string]
			scientistResult, err = scientistStep.Start(ctx, manager.GetMessagesWithSystemPrompt())
			cobra.CheckErr(err)
			mergeResult := steps.Bind[string, []*context2.Message](ctx, scientistResult, mergeStep)
			writerResult := steps.Bind[[]*context2.Message, string](ctx, mergeResult, writerStep)

			res := writerResult.Return()
			_ = res

			return nil
		})

		errgrp.Go(func() error {
			return router.Run(ctx)
		})

		err = errgrp.Wait()
		cobra.CheckErr(err)
	},
}

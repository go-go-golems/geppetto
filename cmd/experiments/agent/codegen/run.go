package codegen

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/steps/utils"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
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
	stepSettings, err := settings.NewStepSettings()
	if err != nil {
		return nil, err
	}
	geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings)
	if err != nil {
		return nil, err
	}

	layers_ := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))

	cobraParser, err := cli.NewCobraParserFromLayers(
		layers_,
		cli.WithCobraMiddlewaresFunc(
			cmds.GetCobraCommandGeppettoMiddlewares,
		))
	cobra.CheckErr(err)

	parsedLayers, err := cobraParser.Parse(cmd, nil)
	cobra.CheckErr(err)

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
			chat.StepPrinterFunc("Scientist", cmd.OutOrStdout()),
		)
		router.AddNoPublisherHandler("writer",
			"writer",
			pubSub,
			chat.StepPrinterFunc("Writer", cmd.OutOrStdout()),
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
			chat.WithPublishedTopic(pubSub, "scientist"),
		)
		cobra.CheckErr(err)
		writerStep, err := writerCommand.CreateStep(
			chat.WithPublishedTopic(pubSub, "writer"),
		)
		cobra.CheckErr(err)

		mergeStep := utils.NewMergeStep(writerManager, true)

		ctx := cmd.Context()

		errgrp := errgroup.Group{}
		errgrp.Go(func() error {
			var scientistResult steps.StepResult[string]
			scientistResult, err = scientistStep.Start(ctx, manager.GetConversation())
			cobra.CheckErr(err)
			mergeResult := steps.Bind[string, conversation.Conversation](ctx, scientistResult, mergeStep)
			writerResult := steps.Bind[conversation.Conversation, string](ctx, mergeResult, writerStep)

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

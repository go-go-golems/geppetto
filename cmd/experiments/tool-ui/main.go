package main

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/charmbracelet/bubbletea"
	boba_chat "github.com/go-go-golems/bobatea/pkg/chat"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/ui"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	glazed_settings "github.com/go-go-golems/glazed/pkg/settings"
	"github.com/invopop/jsonschema"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
)

type ToolUiCommand struct {
	*glazed_cmds.CommandDescription
	stepSettings *settings.StepSettings
	manager      conversation.Manager
	logger       watermill.LoggerAdapter
	pubSub       *gochannel.GoChannel
	router       *message.Router
	reflector    *jsonschema.Reflector
	chatToolStep *openai.ChatToolStep
}

var _ glazed_cmds.GlazeCommand = (*ToolUiCommand)(nil)

func NewToolUiCommand() (*ToolUiCommand, error) {
	glazedParameterLayer, err := glazed_settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}

	stepSettings := settings.NewStepSettings()
	geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings)
	if err != nil {
		return nil, err
	}

	return &ToolUiCommand{
		CommandDescription: glazed_cmds.NewCommandDescription(
			"tool-ui",
			glazed_cmds.WithShort("Tool UI"),
			glazed_cmds.WithFlags(
				parameters.NewParameterDefinition(
					"ui",
					parameters.ParameterTypeBool,
					parameters.WithDefault(false),
					parameters.WithHelp("start in UI mode")),
				parameters.NewParameterDefinition(
					"print-raw-events",
					parameters.ParameterTypeBool,
					parameters.WithDefault(false),
					parameters.WithHelp("print raw events")),
			),
			glazed_cmds.WithLayersList(glazedParameterLayer),
			glazed_cmds.WithLayersList(geppettoLayers...),
		),
	}, nil

}

type ToolUiSettings struct {
	UI             bool `glazed.parameter:"ui"`
	PrintRawEvents bool `glazed.parameter:"print-raw-events"`
}

func (t *ToolUiCommand) RunIntoGlazeProcessor(
	ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor,
) error {
	s := &ToolUiSettings{}

	err := parsedLayers.InitializeStruct(layers.DefaultSlug, s)
	if err != nil {
		return err
	}

	if s.UI {
		return t.runWithUi(ctx, parsedLayers)
	}

	err = t.Init(parsedLayers)
	if err != nil {
		return err
	}

	defer func(pubSub *gochannel.GoChannel) {
		err := pubSub.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close pubSub")
		}
	}(t.pubSub)

	t.router.AddNoPublisherHandler("ui-stdout",
		"ui",
		t.pubSub,
		chat.StepPrinterFunc("UI", os.Stdout),
	)

	ctx, cancel := context.WithCancel(ctx)

	eg := errgroup.Group{}
	eg.Go(func() error {
		defer cancel()

		result, err := t.chatToolStep.Start(ctx, t.manager.GetConversation())
		if err != nil {
			return err
		}
		res := <-result.GetChannel()
		fmt.Printf("\n\nchatToolStep.Start returned %v\n", res.ValueOr("error"))
		return nil
	})

	eg.Go(func() error {
		ret := t.router.Run(ctx)
		fmt.Printf("router.Run returned %v\n", ret)
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

// TODO(manuel, 2024-01-13) Turn this into ToolUiCommand fields and Init method
func (t *ToolUiCommand) Init(parsedLayers *layers.ParsedLayers) error {
	t.stepSettings = settings.NewStepSettings()
	err := t.stepSettings.UpdateFromParsedLayers(parsedLayers)
	if err != nil {
		return err
	}

	t.stepSettings.Chat.Stream = true

	t.manager = conversation.NewManager(
		conversation.WithMessages(
			conversation.NewChatMessage(
				conversation.RoleUser,
				"Give me the weather in Boston on november 9th 1924, please, including the windspeed for me, an old ass american. Also, the weather in paris today, with temperature.",
			),
		))

	t.logger = watermill.NopLogger{}
	t.pubSub = gochannel.NewGoChannel(gochannel.Config{
		BlockPublishUntilSubscriberAck: true,
	}, t.logger)

	t.router, err = message.NewRouter(message.RouterConfig{}, t.logger)
	if err != nil {
		return err
	}

	t.reflector = &jsonschema.Reflector{
		DoNotReference: true,
	}
	err = t.reflector.AddGoComments("github.com/go-go-golems/geppetto", "./cmd/experiments/tool-ui")
	if err != nil {
		log.Warn().Err(err).Msg("Could not add go comments")
	}

	t.chatToolStep, err = openai.NewChatToolStep(
		t.stepSettings,
		openai.WithReflector(t.reflector),
		openai.WithToolFunctions(map[string]interface{}{
			"getWeather":      getWeather,
			"getWeatherOnDay": getWeatherOnDay,
		}),
	)
	if err != nil {
		return err
	}
	err = t.chatToolStep.AddPublishedTopic(t.pubSub, "ui")
	if err != nil {
		return err
	}

	return nil
}

func (t *ToolUiCommand) runWithUi(ctx context.Context,
	parsedLayers *layers.ParsedLayers,
) error {
	err := t.Init(parsedLayers)
	if err != nil {
		return err
	}

	backend := ui.NewStepBackend(t.chatToolStep)

	// Create bubbletea UI

	options := []tea.ProgramOption{
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	}
	options = append(options, tea.WithAltScreen())

	// maybe test with CLI output first

	p := tea.NewProgram(
		boba_chat.InitialModel(t.manager, backend),
		options...,
	)
	_ = p

	t.router.AddNoPublisherHandler("ui",
		"ui", t.pubSub,
		func(msg *message.Message) error {
			msg.Ack()

			e, err := chat.NewEventFromJson(msg.Payload)
			if err != nil {
				return err
			}

			metadata := boba_chat.StreamMetadata{
				ID:             e.Metadata.ID,
				ParentID:       e.Metadata.ParentID,
				ConversationID: e.Metadata.ConversationID,
			}

			switch e.Type {
			case chat.EventTypeError:
				p.Send(boba_chat.StreamCompletionError{
					Err:            e.Error,
					StreamMetadata: metadata,
				})
			case chat.EventTypePartial:
				p_, ok := e.ToPartialCompletion()
				if !ok {
					return errors.New("payload is not of type EventPartialCompletionPayload")
				}
				p.Send(boba_chat.StreamCompletionMsg{
					Delta:          p_.Delta,
					Completion:     p_.Completion,
					StreamMetadata: metadata,
				})
			case chat.EventTypeFinal:
				p_, ok := e.ToText()
				if !ok {
					return errors.New("payload is not of type EventTextPayload")
				}
				p.Send(boba_chat.StreamDoneMsg{
					StreamMetadata: metadata,
					Completion:     p_.Text,
				})
			case chat.EventTypeInterrupt:
				p.Send(boba_chat.StreamDoneMsg{
					StreamMetadata: metadata,
				})
			case chat.EventTypeStart:
				p.Send(boba_chat.StreamStartMsg{
					StreamMetadata: metadata,
				})
			case chat.EventTypeStatus:
				p_, ok := e.ToText()
				if !ok {
					return errors.New("payload is not of type EventTextPayload")
				}
				p.Send(boba_chat.StreamStatusMsg{
					Text:           p_.Text,
					StreamMetadata: metadata,
				})
			}

			_ = metadata

			return nil
		})

	ctx, cancel := context.WithCancel(ctx)

	eg := errgroup.Group{}

	eg.Go(func() error {
		ret := t.router.Run(ctx)
		fmt.Printf("router.Run returned %v\n", ret)
		return nil
	})

	eg.Go(func() error {
		if _, err := p.Run(); err != nil {
			return err
		}
		defer cancel()
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	return &glazed_cmds.ExitWithoutGlazeError{}
}

func main() {
	toolUiCommand, err := NewToolUiCommand()
	cobra.CheckErr(err)

	toolUICobraCommand, err := cmds.BuildCobraCommandWithGeppettoMiddlewares(toolUiCommand)
	cobra.CheckErr(err)

	err = clay.InitViper("pinocchio", toolUICobraCommand)
	cobra.CheckErr(err)

	err = toolUICobraCommand.Execute()
	cobra.CheckErr(err)
}

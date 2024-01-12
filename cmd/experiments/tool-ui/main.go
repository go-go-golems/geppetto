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
			),
			glazed_cmds.WithLayersList(glazedParameterLayer),
			glazed_cmds.WithLayersList(geppettoLayers...),
		),
	}, nil

}

type ToolUiSettings struct {
	UI bool `glazed.parameter:"ui"`
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

	router, pubSub, manager, chatToolStep, err := t.setup(parsedLayers)
	if err != nil {
		return err
	}

	defer func(pubSub *gochannel.GoChannel) {
		err := pubSub.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close pubSub")
		}
	}(pubSub)

	router.AddNoPublisherHandler("ui-stdout",
		"ui",
		pubSub,
		func(msg *message.Message) error {
			msg.Ack()

			e, err := chat.NewEventFromJson(msg.Payload)
			if err != nil {
				return err
			}

			switch e.Type {
			case chat.EventTypeError:
				return err
			case chat.EventTypePartial:
				p_, ok := e.ToPartialCompletion()
				if !ok {
					return errors.Errorf("Invalid payload type")
				}
				_, err = os.Stderr.Write([]byte(p_.Delta))
				if err != nil {
					return err
				}
			case chat.EventTypeFinal:
			case chat.EventTypeInterrupt:
			}

			return nil
		})

	ctx, cancel := context.WithCancel(ctx)

	eg := errgroup.Group{}
	eg.Go(func() error {
		defer cancel()

		result, err := chatToolStep.Start(ctx, manager.GetMessages())
		if err != nil {
			return err
		}
		res := <-result.GetChannel()
		fmt.Printf("\n\nchatToolStep.Start returned %v\n", res.ValueOr("error"))
		return nil
	})

	eg.Go(func() error {
		ret := router.Run(ctx)
		fmt.Printf("router.Run returned %v\n", ret)
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (t *ToolUiCommand) setup(parsedLayers *layers.ParsedLayers) (
	*message.Router, *gochannel.GoChannel, *conversation.Manager, *openai.ChatToolStep, error,
) {
	stepSettings := settings.NewStepSettings()

	err := stepSettings.UpdateFromParsedLayers(parsedLayers)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	stepSettings.Chat.Stream = true

	manager := conversation.NewManager(
		conversation.WithMessages(
			[]*conversation.Message{
				conversation.NewMessage(
					"Give me the weather in Boston on november 9th 1924, please, including the windspeed for me, an old ass american. Also, the weather in paris today, with temperature.",
					conversation.RoleUser,
				),
			}))

	logger := watermill.NopLogger{}
	pubSub := gochannel.NewGoChannel(gochannel.Config{
		// Guarantee that messages are delivered in the order of publishing.
		BlockPublishUntilSubscriberAck: true,
	}, logger)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	reflector := &jsonschema.Reflector{
		DoNotReference: true,
	}
	err = reflector.AddGoComments("github.com/go-go-golems/geppetto", "./cmd/experiments/tool-ui")
	if err != nil {
		log.Warn().Err(err).Msg("Could not add go comments")
	}

	chatToolStep, err := openai.NewChatToolStep(
		stepSettings,
		openai.WithReflector(reflector),
		openai.WithToolFunctions(map[string]interface{}{
			"getWeather":      getWeather,
			"getWeatherOnDay": getWeatherOnDay,
		}),
	)
	err = chatToolStep.AddPublishedTopic(pubSub, "ui")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return router, pubSub, manager, chatToolStep, nil
}

func (t *ToolUiCommand) runWithUi(ctx context.Context,
	parsedLayers *layers.ParsedLayers,
) error {
	router, pubSub, manager, chatToolStep, err := t.setup(parsedLayers)
	if err != nil {
		return err
	}

	backend := ui.NewStepBackend(chatToolStep)

	// Create bubbletea UI

	options := []tea.ProgramOption{
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	}
	options = append(options, tea.WithAltScreen())

	// maybe test with CLI output first

	p := tea.NewProgram(
		boba_chat.InitialModel(manager, backend),
		options...,
	)
	_ = p

	router.AddNoPublisherHandler("ui",
		"ui", pubSub,
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
		ret := router.Run(ctx)
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

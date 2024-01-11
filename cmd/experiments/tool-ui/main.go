package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	tea "github.com/charmbracelet/bubbletea"
	boba_chat "github.com/go-go-golems/bobatea/pkg/chat"
	"github.com/go-go-golems/bobatea/pkg/chat/conversation"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/ui"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/invopop/jsonschema"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
)

type WeatherData struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	WindSpeed   float64 `json:"wind_speed"`
}

type WeatherRequest struct {
	// The city for which to request the data
	City string `json:"city"`
	// Return windspeed in km/h
	WindSpeed bool `json:"wind_speed"`
	// Return temperature in Celsius
	Temperature bool `json:"temperature"`
}

func getWeather(request WeatherRequest) WeatherData {
	return WeatherData{
		City:        request.City,
		Temperature: 23.0,
		WindSpeed:   10.0,
	}
}

type WeatherOnDayRequest struct {
	WeatherRequest
	// The date for which to request the data
	Date string `json:"date"`
}

func getWeatherOnDay(request WeatherOnDayRequest) WeatherData {
	return WeatherData{
		City:        request.City,
		Temperature: 23.0,
		WindSpeed:   10.0,
	}
}

var ToolUiCmd = &cobra.Command{
	Use:   "tool-ui",
	Short: "Tool UI",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(cmd.Context())

		stepSettings := settings.NewStepSettings()
		geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings)
		cobra.CheckErr(err)
		layers_ := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))

		cobraParser, err := cli.NewCobraParserFromLayers(
			layers_,
			cli.WithCobraMiddlewaresFunc(
				cmds.GetCobraCommandGeppettoMiddlewares,
			))
		cobra.CheckErr(err)

		parsedLayers, err := cobraParser.Parse(cmd, args)
		cobra.CheckErr(err)

		err = stepSettings.UpdateFromParsedLayers(parsedLayers)
		cobra.CheckErr(err)

		stepSettings.Chat.Stream = true

		manager := conversation.NewManager(conversation.WithMessages([]*conversation.Message{
			conversation.NewMessage(
				"Give me the weather in Boston on november 9th 1924, please, including the windspeed for me, an old ass american. Also, the weather in paris today, with temperature.",
				conversation.RoleUser,
			),
		}))

		// Configure pubsub

		logger := watermill.NopLogger{}
		pubSub := gochannel.NewGoChannel(gochannel.Config{
			// Guarantee that messages are delivered in the order of publishing.
			BlockPublishUntilSubscriberAck: true,
		}, logger)

		router, err := message.NewRouter(message.RouterConfig{}, logger)
		cobra.CheckErr(err)

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
				e := &chat.Event{}
				err := json.Unmarshal(msg.Payload, e)
				if err != nil {
					return err
				}

				switch e.Type {
				case chat.EventTypeError:
					return err
				case chat.EventTypePartial:
					_, err = os.Stderr.Write([]byte(e.Text))
					if err != nil {
						return err
					}
				case chat.EventTypeFinal:
				case chat.EventTypeInterrupt:
				}

				msg.Ack()

				return nil
			})

		eg := errgroup.Group{}
		// Create toolStep

		reflector := &jsonschema.Reflector{
			DoNotReference: true,
		}
		err = reflector.AddGoComments("github.com/go-go-golems/geppetto", "./cmd/tool-ui/main")
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
		err = chatToolStep.Publish(pubSub, "ui")
		cobra.CheckErr(err)

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
				e := &chat.Event{}
				err := json.Unmarshal(msg.Payload, e)
				if err != nil {
					return err
				}

				metadata := boba_chat.StreamMetadata{
					ID:             e.Metadata.ID,
					ParentID:       e.Metadata.ParentID,
					ConversationID: e.Metadata.ConversationID,
				}
				//
				//		switch e.Type {
				//		case chat.EventTypeError:
				//			p.Send(boba_chat.StreamCompletionError{
				//				Err:            e.Error,
				//				StreamMetadata: metadata,
				//			})
				//		case chat.EventTypePartial:
				//			p.Send(boba_chat.StreamCompletionMsg{
				//				Completion:     e.Text,
				//				StreamMetadata: metadata,
				//			})
				//		case chat.EventTypeFinal:
				//			p.Send(boba_chat.StreamDoneMsg{
				//				StreamMetadata: metadata,
				//			})
				//		case chat.EventTypeInterrupt:
				//			p.Send(boba_chat.StreamDoneMsg{
				//				StreamMetadata: metadata,
				//			})
				//		case chat.EventTypeStart:
				//			p.Send(boba_chat.StreamStartMsg{
				//				StreamMetadata: metadata,
				//			})
				//		case chat.EventTypeStatus:
				//			p.Send(boba_chat.StreamStatusMsg{
				//				Text:           e.Text,
				//				StreamMetadata: metadata,
				//			})
				//		}
				//
				fmt.Println("got event", e)
				_ = metadata
				msg.Ack()

				return nil
			})

		eg.Go(func() error {
			ret := router.Run(ctx)
			fmt.Printf("router.Run returned %v\n", ret)
			return nil
		})

		eg.Go(func() error {
			result, err := chatToolStep.Start(ctx, manager.GetMessages())
			if err != nil {
				return err
			}
			res := <-result.GetChannel()
			fmt.Printf("chatToolStep.Start returned %v\n", res.ValueOr("error"))
			//if _, err := p.Run(); err != nil {
			//	return err
			//}
			defer cancel()
			return nil
		})

		err = eg.Wait()
		cobra.CheckErr(err)

	},
}

func main() {
	stepSettings := settings.NewStepSettings()
	geppettoLayers, err := cmds.CreateGeppettoLayers(stepSettings)
	cobra.CheckErr(err)

	pLayers := layers.NewParameterLayers(layers.WithLayers(geppettoLayers...))
	err = pLayers.AddToCobraCommand(ToolUiCmd)
	err = clay.InitViper("pinocchio", ToolUiCmd)
	cobra.CheckErr(err)

	err = ToolUiCmd.Execute()
	cobra.CheckErr(err)
}

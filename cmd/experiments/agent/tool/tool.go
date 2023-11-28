package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/geppetto/cmd/experiments/agent/helpers"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openai2 "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/invopop/jsonschema"
	"github.com/rs/zerolog/log"
	go_openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
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

var ToolCallCmd = &cobra.Command{
	Use:   "tool-call",
	Short: "Tool call",
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := openai2.NewParameterLayer()
		cobra.CheckErr(err)
		aiLayer, err := settings.NewChatParameterLayer()
		cobra.CheckErr(err)

		// TODO(manuel, 2023-11-28) Turn this into a "add all flags to command"
		// function to create commands, like glazedParameterLayer
		parsedLayers, err := helpers.ParseLayersFromCobraCommand(cmd, []cli.CobraParameterLayer{layer, aiLayer})
		cobra.CheckErr(err)

		stepSettings := settings.NewStepSettings()
		err = stepSettings.UpdateFromParsedLayers(parsedLayers)
		cobra.CheckErr(err)

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		messages := []*geppetto_context.Message{
			{
				Text: "Give me the weather in Boston, please, including the windspeed for me, an old ass american.",
				Role: geppetto_context.RoleUser,
			},
		}

		//
		reflector := new(jsonschema.Reflector)
		err = reflector.AddGoComments("github.com/go-go-golems/geppetto", "./cmd/experiments/agent")
		if err != nil {
			log.Warn().Err(err).Msg("Could not add go comments")
		}
		getWeatherJsonSchema := reflector.Reflect(WeatherRequest{})

		marshal, err := json.MarshalIndent(getWeatherJsonSchema.Definitions["WeatherRequest"], "", "  ")
		cobra.CheckErr(err)
		fmt.Printf("getWeather jsonschema\n:%s\n", marshal)

		// LLM completion step
		step := &openai.ToolStep{
			Settings: stepSettings,
			Tools: []go_openai.Tool{{
				Type: "function",
				Function: go_openai.FunctionDefinition{
					Name:        "getWeather",
					Description: "Get the weather",
					Parameters:  getWeatherJsonSchema.Definitions["WeatherRequest"],
				},
			},
			},
		}

		//step.SetStreaming(true)

		// start the LLM completion
		res, err := step.Start(ctx, messages)
		cobra.CheckErr(err)

		c := res.GetChannel()
		for i := range c {
			s, err := i.Value()
			cobra.CheckErr(err)
			fmt.Printf("%s", s)
		}
	},
}

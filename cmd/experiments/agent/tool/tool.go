package tool

import (
	"context"
	"encoding/json"
	"fmt"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	helpers2 "github.com/go-go-golems/geppetto/pkg/helpers"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	openai2 "github.com/go-go-golems/geppetto/pkg/steps/ai/settings/openai"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
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

var ToolCallCmd = &cobra.Command{
	Use:   "tool-call",
	Short: "Tool call",
	Run: func(cmd *cobra.Command, args []string) {
		layer, err := openai2.NewParameterLayer()
		cobra.CheckErr(err)
		aiLayer, err := settings.NewChatParameterLayer()
		cobra.CheckErr(err)

		layers_ := layers.NewParameterLayers(layers.WithLayers(layer, aiLayer))
		parsedLayers, err := cli.ParseLayersFromCobraCommand(cmd, layers_)
		cobra.CheckErr(err)

		stepSettings := settings.NewStepSettings()
		err = stepSettings.UpdateFromParsedLayers(parsedLayers)
		cobra.CheckErr(err)

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()
		messages := []*geppetto_context.Message{
			{
				Text: "Give me the weather in Boston on november 9th 1924, please, including the windspeed for me, an old ass american. Also, the weather in paris today, with temperature.",
				Role: geppetto_context.RoleUser,
			},
		}

		//
		reflector := new(jsonschema.Reflector)
		err = reflector.AddGoComments("github.com/go-go-golems/geppetto", "./cmd/experiments/agent")
		if err != nil {
			log.Warn().Err(err).Msg("Could not add go comments")
		}
		getWeatherOnDayJsonSchema, _ := helpers2.GetFunctionParametersJsonSchema(getWeatherOnDay)
		s, _ := json.MarshalIndent(getWeatherOnDayJsonSchema, "", " ")
		fmt.Printf("getWeatherOnDayJsonSchema:\n%s\n\n", s)

		getWeatherJsonSchema, _ := helpers2.GetFunctionParametersJsonSchema(getWeather)
		s, _ = json.MarshalIndent(getWeatherJsonSchema, "", " ")
		fmt.Printf("getWeatherJsonSchema:\n%s\n\n", s)

		// LLM completion step
		step := &openai.ToolStep{
			Settings: stepSettings,
			Tools: []go_openai.Tool{{
				Type: "function",
				Function: go_openai.FunctionDefinition{
					Name:        "getWeather",
					Description: "Get the weather",
					Parameters:  getWeatherJsonSchema,
				},
			},
				{
					Type: "function",
					Function: go_openai.FunctionDefinition{
						Name:        "getWeatherOnDay",
						Description: "Get the weather on a specific day",
						Parameters:  getWeatherOnDayJsonSchema,
					},
				},
			},
		}

		execStep := &openai.ExecuteToolStep{
			Tools: map[string]interface{}{
				"getWeather":      getWeather,
				"getWeatherOnDay": getWeatherOnDay,
			},
		}

		//step.SetStreaming(true)

		// start the LLM completion
		res, err := step.Start(ctx, messages)
		cobra.CheckErr(err)

		res_ := steps.Bind[openai.ToolCompletionResponse, map[string]interface{}](ctx, res, execStep)

		c := res_.GetChannel()
		for i := range c {
			s, err := i.Value()
			cobra.CheckErr(err)

			s_, _ := json.MarshalIndent(s, "", " ")
			fmt.Printf("%s", s_)
		}
	},
}

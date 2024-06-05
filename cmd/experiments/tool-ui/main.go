package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbletea"
	boba_chat "github.com/go-go-golems/bobatea/pkg/chat"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/geppetto/pkg/cmds"
	"github.com/go-go-golems/geppetto/pkg/events"
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
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
)

type ToolUiCommand struct {
	*glazed_cmds.CommandDescription
	stepSettings *settings.StepSettings
	manager      conversation.Manager
	reflector    *jsonschema.Reflector
	chatToolStep *openai.ChatExecuteToolStep
	eventRouter  *events.EventRouter
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
				parameters.NewParameterDefinition(
					"verbose",
					parameters.ParameterTypeBool,
					parameters.WithDefault(false),
					parameters.WithHelp("verbose")),
			),
			glazed_cmds.WithLayersList(glazedParameterLayer),
			glazed_cmds.WithLayersList(geppettoLayers...),
		),
	}, nil

}

type ToolUiSettings struct {
	UI             bool `glazed.parameter:"ui"`
	PrintRawEvents bool `glazed.parameter:"print-raw-events"`
	Verbose        bool `glazed.parameter:"verbose"`
}

func (t *ToolUiCommand) RunIntoGlazeProcessor(
	ctx context.Context, parsedLayers *layers.ParsedLayers, gp middlewares.Processor,
) error {
	settings := &ToolUiSettings{}

	err := parsedLayers.InitializeStruct(layers.DefaultSlug, settings)
	if err != nil {
		return err
	}

	if settings.UI {
		return t.runWithUi(ctx, parsedLayers)
	}

	err = t.Init(parsedLayers)
	if err != nil {
		return err
	}

	defer func() {
		err := t.eventRouter.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close eventRouter")
		}
	}()

	if settings.PrintRawEvents {
		t.eventRouter.AddHandler("raw-events-stdout", "ui", t.eventRouter.DumpRawEvents)
	} else {
		t.eventRouter.AddHandler("ui-stdout",
			"ui",
			chat.StepPrinterFunc("UI", os.Stdout),
		)
	}

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
		ret := t.eventRouter.Run(ctx)
		fmt.Printf("eventRouter.Run returned %v\n", ret)
		return nil
	})

	err = eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

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

	t.eventRouter, err = events.NewEventRouter()
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
	err = t.chatToolStep.AddPublishedTopic(t.eventRouter.Publisher, "ui")
	if err != nil {
		return err
	}

	return nil
}

func (t *ToolUiCommand) runWithUi(
	ctx context.Context,
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

	t.eventRouter.AddHandler("ui", "ui", ui.StepChatForwardFunc(p))

	ctx, cancel := context.WithCancel(ctx)

	eg := errgroup.Group{}

	eg.Go(func() error {
		ret := t.eventRouter.Run(ctx)
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

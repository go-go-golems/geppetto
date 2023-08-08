package cmds

import (
	"context"
	_ "embed"
	"fmt"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/openai"
	"github.com/go-go-golems/geppetto/pkg/steps/openai/completion"
	glazedcmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"time"
)

type GeppettoCommandDescription struct {
	Name      string                            `yaml:"name"`
	Short     string                            `yaml:"short"`
	Long      string                            `yaml:"long,omitempty"`
	Flags     []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
	Arguments []*parameters.ParameterDefinition `yaml:"arguments,omitempty"`
	Layers    []layers.ParameterLayer           `yaml:"layers,omitempty"`

	// TODO(manuel, 2023-02-04) This now has a hack to switch the step type
	Step *steps.StepDescription `yaml:"step,omitempty"`

	Prompt       string                      `yaml:"prompt,omitempty"`
	Messages     []*geppetto_context.Message `yaml:"messages,omitempty"`
	SystemPrompt string                      `yaml:"system-prompt,omitempty"`
}

func HelpersParameterLayer() (layers.ParameterLayer, error) {
	return layers.NewParameterLayer("helpers", "Geppetto helpers",
		layers.WithFlags(
			parameters.NewParameterDefinition(
				"print-prompt",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Print the prompt"),
			),
			parameters.NewParameterDefinition(
				"print-dyno",
				parameters.ParameterTypeBool,
				parameters.WithDefault(false),
				parameters.WithHelp("Print the dyno embed div"),
			),
			parameters.NewParameterDefinition(
				"system",
				parameters.ParameterTypeString,
				parameters.WithHelp("System message"),
			),
			parameters.NewParameterDefinition(
				"append-message-file",
				parameters.ParameterTypeString,
				parameters.WithHelp("File containing messages (json or yaml, list of objects with fields text, time, role) to be appended to the already present list of messages"),
			),
			parameters.NewParameterDefinition(
				"message-file",
				parameters.ParameterTypeString,
				parameters.WithHelp("File containing messages (json or yaml, list of objects with fields text, time, role)"),
			),
		),
	)
}

type GeppettoCommand struct {
	*glazedcmds.CommandDescription
	Factories    map[string]interface{} `yaml:"__factories,omitempty"`
	Prompt       string
	Messages     []*geppetto_context.Message
	SystemPrompt string
}

type GeppettoCommandOption func(*GeppettoCommand)

func WithPrompt(prompt string) GeppettoCommandOption {
	return func(g *GeppettoCommand) {
		g.Prompt = prompt
	}
}

func WithMessages(messages []*geppetto_context.Message) GeppettoCommandOption {
	return func(g *GeppettoCommand) {
		g.Messages = messages
	}
}

func WithSystemPrompt(systemPrompt string) GeppettoCommandOption {
	return func(g *GeppettoCommand) {
		g.SystemPrompt = systemPrompt
	}
}

func NewGeppettoCommand(
	description *glazedcmds.CommandDescription,
	factories map[string]interface{},
	options ...GeppettoCommandOption,
) (*GeppettoCommand, error) {
	helpersParameterLayer, err := HelpersParameterLayer()
	if err != nil {
		return nil, err
	}

	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}

	description.Layers = append(
		description.Layers,
		helpersParameterLayer,
		glazedParameterLayer)

	ret := &GeppettoCommand{
		CommandDescription: description,
		Factories:          factories,
	}

	for _, option := range options {
		option(ret)
	}

	return ret, nil
}

//go:embed templates/dyno.tmpl.html
var dynoTemplate string

func (g *GeppettoCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	if g.Prompt != "" && len(g.Messages) != 0 {
		return errors.Errorf("Prompt and messages are mutually exclusive")
	}

	for _, f := range g.Factories {
		factory, ok := f.(steps.GenericStepFactory)
		if !ok {
			continue
		}
		err := factory.UpdateFromParameters(ps)
		if err != nil {
			return err
		}
	}

	messages := g.Messages
	systemPrompt := g.SystemPrompt

	contextManager := geppetto_context.NewManager()

	systemPrompt_, ok := ps["system"].(string)
	if ok && systemPrompt_ != "" {
		systemPrompt = systemPrompt_
	}

	messageFile, ok := ps["message-file"].(string)
	if ok && messageFile != "" {
		messages_, err := geppetto_context.LoadFromFile(messageFile)
		if err != nil {
			return err
		}
		messages = messages_
	}

	appendMessageFile, ok := ps["append-message-file"].(string)
	if ok && appendMessageFile != "" {
		messages_, err := geppetto_context.LoadFromFile(appendMessageFile)
		if err != nil {
			return err
		}
		messages = append(messages, messages_...)
	}

	// NOTE(manuel, 2023-07-21)
	// This is called completion, but it actually branches out to the chat API.'
	// Maybe in the future we can let this slide entirely, in fact this should be replaced by a single step
	// in a program, so this will probably be refactored away.
	// In fact, this might already get all streamlined once I redo the monad/step part.
	openaiCompletionStepFactory_, ok := g.Factories["openai-completion-step"]
	if !ok {
		return errors.Errorf("No openai-completion-step factory defined")
	}
	openaiCompletionStepFactory, ok := openaiCompletionStepFactory_.(completion.CompletionStepFactory)
	if !ok {
		return errors.Errorf("openai-completion-step factory is not a StepFactory[string, string]")
	}

	s, err := openaiCompletionStepFactory.NewStep()
	if err != nil {
		return err
	}

	if systemPrompt != "" {
		systemPromptTemplate, err := templating.CreateTemplate("system-prompt").Parse(systemPrompt)
		if err != nil {
			return err
		}

		var systemPromptBuffer strings.Builder
		err = systemPromptTemplate.Execute(&systemPromptBuffer, ps)
		if err != nil {
			return err
		}

		contextManager.SetSystemPrompt(systemPromptBuffer.String())
	}

	if len(messages) > 0 {
		for _, message := range messages {
			messageTemplate, err := templating.CreateTemplate("message").Parse(message.Text)
			if err != nil {
				return err
			}

			var messageBuffer strings.Builder
			err = messageTemplate.Execute(&messageBuffer, ps)
			if err != nil {
				return err
			}
			s_ := messageBuffer.String()

			contextManager.AddMessages(&geppetto_context.Message{
				Text: s_,
				Role: message.Role,
				Time: message.Time,
			})
		}
	}

	if g.Prompt != "" {
		// TODO(manuel, 2023-02-04) All this could be handle by some prompt renderer kind of thing
		promptTemplate, err := templating.CreateTemplate("prompt").Parse(g.Prompt)
		if err != nil {
			return err
		}

		// TODO(manuel, 2023-02-04) This is where multisteps would work differently, since
		// the prompt would be rendered at execution time
		var promptBuffer strings.Builder
		err = promptTemplate.Execute(&promptBuffer, ps)
		if err != nil {
			return err
		}

		contextManager.AddMessages(&geppetto_context.Message{
			Text: promptBuffer.String(),
			Role: "user",
			Time: time.Now(),
		})
	}

	printPrompt, ok := ps["print-prompt"]
	if ok && printPrompt.(bool) {
		fmt.Println(contextManager.GetSinglePrompt())
		return nil
	}

	printDyno, ok := ps["print-dyno"]
	if ok && printDyno.(bool) {
		openaiCompletionStepFactory__, ok := openaiCompletionStepFactory_.(*completion.StepFactory)
		if !ok {
			return errors.Errorf("openai-completion-step factory is not a StepFactory")
		}
		settings := openaiCompletionStepFactory__.StepSettings

		dyno, err := templating.RenderHtmlTemplateString(dynoTemplate, map[string]interface{}{
			// TODO(manuel, 2023-07-21) Does dyno support the chat API now?
			"initialPrompt":   contextManager.GetSinglePrompt(),
			"initialResponse": "",
			"maxTokens":       settings.MaxResponseTokens,
			"temperature":     settings.Temperature,
			"topP":            settings.TopP,
			"model":           settings.Engine,
		})
		if err != nil {
			return err
		}
		fmt.Println(dyno)
		return nil
	}

	eg, ctx2 := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return s.Run(ctx2, contextManager)
	})

	accumulate := ""

	openAILayer, ok := parsedLayers["openai-completion"]
	if !ok {
		return errors.Errorf("No openai layer")
	}
	isStream := openAILayer.Parameters["openai-stream"].(bool)
	log.Debug().Bool("isStream", isStream).Msg("")

	eg.Go(func() error {
		for {
			select {
			case <-ctx2.Done():
				return ctx2.Err()
			case result := <-s.GetOutput():
				if !result.Ok() {
					return result.Error()
				}

				v, err := result.Value()
				if err != nil {
					return err
				}

				if result.IsPartial() {
					fmt.Print(v)
					accumulate += v
				} else {
					if !isStream {
						err = gp.AddRow(ctx, types.NewRow(
							types.MRP("response", accumulate+v),
						))
					}

					if err != nil {
						return err
					}
					return nil

				}
			}

		}
	})
	return eg.Wait()
}

type GeppettoCommandLoader struct {
}

func (g *GeppettoCommandLoader) LoadCommandFromYAML(
	s io.Reader,
	options ...glazedcmds.CommandDescriptionOption,
) ([]glazedcmds.Command, error) {
	yamlContent, err := io.ReadAll(s)
	if err != nil {
		return nil, err
	}

	buf := strings.NewReader(string(yamlContent))
	scd := &GeppettoCommandDescription{
		Flags:     []*parameters.ParameterDefinition{},
		Arguments: []*parameters.ParameterDefinition{},
	}
	err = yaml.NewDecoder(buf).Decode(scd)
	if err != nil {
		return nil, err
	}

	// TODO(manuel, 2023-01-27): There has to be a better way to parse YAML factories
	// maybe the easiest is just going to be to make them a separate file in the bundle format, really
	// rewind to read the factories...
	buf = strings.NewReader(string(yamlContent))
	completionStepFactory, err := completion.NewStepFactoryFromYAML(buf)
	if err != nil {
		return nil, err
	}

	// check if the openai-api-key is set in viper
	openaiAPIKey := viper.GetString("openai-api-key")
	if openaiAPIKey != "" {
		completionStepFactory.ClientSettings.APIKey = &openaiAPIKey
	}

	completionParameterLayer, err := completion.NewParameterLayer(
		layers.WithDefaults(completionStepFactory.StepSettings),
	)
	if err != nil {
		return nil, err
	}

	clientParameterLayer, err := openai.NewClientParameterLayer(
		layers.WithDefaults(completionStepFactory.ClientSettings),
	)
	if err != nil {
		return nil, err
	}

	ls := append(scd.Layers, completionParameterLayer, clientParameterLayer)

	factories := map[string]interface{}{}
	if completionStepFactory != nil {
		factories["openai-completion-step"] = completionStepFactory
	}

	options_ := []glazedcmds.CommandDescriptionOption{
		glazedcmds.WithShort(scd.Short),
		glazedcmds.WithLong(scd.Long),
		glazedcmds.WithFlags(scd.Flags...),
		glazedcmds.WithArguments(scd.Arguments...),
		glazedcmds.WithLayers(ls...),
	}

	description := glazedcmds.NewCommandDescription(
		scd.Name,
		options_...,
	)
	if scd.Prompt != "" && len(scd.Messages) != 0 {
		return nil, errors.Errorf("Prompt and messages are mutually exclusive")
	}

	sq, err := NewGeppettoCommand(description, factories,
		WithPrompt(scd.Prompt),
		WithMessages(scd.Messages),
		WithSystemPrompt(scd.SystemPrompt),
	)
	if err != nil {
		return nil, err
	}

	for _, option := range options {
		option(sq.Description())
	}

	return []glazedcmds.Command{sq}, nil
}

func (g *GeppettoCommandLoader) LoadCommandAliasFromYAML(s io.Reader, options ...alias.Option) ([]*alias.CommandAlias, error) {
	return loaders.LoadCommandAliasFromYAML(s, options...)
}

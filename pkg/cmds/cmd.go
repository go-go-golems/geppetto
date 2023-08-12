package cmds

import (
	"context"
	_ "embed"
	"fmt"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/openai/chat"
	glazedcmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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

	description.Layers = append(
		description.Layers,
		helpersParameterLayer,
	)

	ret := &GeppettoCommand{
		CommandDescription: description,
		Factories:          factories,
	}

	for _, option := range options {
		option(ret)
	}

	return ret, nil
}

func (g *GeppettoCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	w io.Writer,
) error {
	if g.Prompt != "" && len(g.Messages) != 0 {
		return errors.Errorf("Prompt and messages are mutually exclusive")
	}

	for _, f := range g.Factories {
		factory, ok := f.(*chat.Transformer)
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
	transformer_, ok := g.Factories["openai-chat"]
	if !ok {
		return errors.Errorf("No openai-completion-step factory defined")
	}
	transformer, ok := transformer_.(steps.Step[[]*geppetto_context.Message, string])
	if !ok {
		return errors.Errorf("openai-completion-step factory is not a StepFactory[string, string]")
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

	messagesM := steps.Resolve(contextManager.GetMessagesWithSystemPrompt())
	m := steps.Bind[[]*geppetto_context.Message, string](ctx, messagesM, transformer)

	accumulate := ""

	openAILayer, ok := parsedLayers["openai-completion"]
	if !ok {
		return errors.Errorf("No openai layer")
	}
	isStream := openAILayer.Parameters["openai-stream"].(bool)
	log.Debug().Bool("isStream", isStream).Msg("")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case result, ok := <-m.GetChannel():
			if !ok {
				return nil
			}

			if !result.Ok() {
				return result.Error()
			}

			v, err := result.Value()
			if err != nil {
				return err
			}

			_, err = w.Write([]byte(v))
			if err != nil {
				return err
			}
			accumulate += v
		}
	}
}

package cmds

import (
	"context"
	_ "embed"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	geppetto_context "github.com/go-go-golems/geppetto/pkg/context"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/openai/chat"
	"github.com/go-go-golems/geppetto/pkg/ui"
	glazedcmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/tcnksm/go-input"
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

// RunIntoWriter runs the command and writes the output into the given writer.
// It first:
//   - configures the factories with the given parameters (for example to override the
//     temperature or other openai settings)
//   - configures the context manager with the configured Messages and SystemPrompt
//     (per default, from the command definition, but can be overloaded through the
//     --system, --message-file and --append-message-file flags)
//   - if --print-prompt is given, it prints the prompt and exits
//
// It then instantiates a
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
		factory, ok := f.(*chat.Step)
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

	// load and render the system prompt
	systemPrompt_, ok := ps["system"].(string)
	if ok && systemPrompt_ != "" {
		systemPrompt = systemPrompt_
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

	// load and render messages
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

	// render the prompt
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

	chatStep_, ok := g.Factories["openai-chat"]
	if !ok {
		return errors.Errorf("No openai-chat-step factory defined")
	}
	chatStep, ok := chatStep_.(steps.Step[[]*geppetto_context.Message, string])
	if !ok {
		return errors.Errorf("openai-chat-step factory is not a StepFactory[string, string]")
	}

	m := steps.Bind[[]*geppetto_context.Message, string](ctx, messagesM, chatStep)

	openAILayer, ok := parsedLayers["openai-chat"]
	if !ok {
		return errors.Errorf("No openai layer")
	}
	isStream := openAILayer.Parameters["openai-stream"].(bool)
	log.Debug().Bool("isStream", isStream).Msg("")

	accumulate, err := g.readStepResults(ctx, m, w)
	if err != nil {
		return err
	}
	contextManager.AddMessages(&geppetto_context.Message{
		Role: "assistant",
		Time: time.Now(),
		Text: accumulate,
	})

	interactive := true
	if interactive {
		tty_, err := ui.OpenTTY()
		if err != nil {
			return err
		}
		defer func() {
			err := tty_.Close()
			if err != nil {
				fmt.Println("Failed to close tty:", err)
			}
		}()

		ui := &input.UI{
			Writer: tty_,
			Reader: tty_,
		}

		if !strings.HasSuffix(accumulate, "\n") {
			fmt.Println()
		}

		query := "\nDo you want to continue in chat? [y/n]"
		answer, err := ui.Ask(query, &input.Options{
			Default:  "y",
			Required: true,
			Loop:     true,
			ValidateFunc: func(answer string) error {
				switch answer {
				case "y", "Y", "n", "N":
					return nil
				default:
					return fmt.Errorf("please enter 'y' or 'n'")
				}
			},
		})

		if err != nil {
			fmt.Println("Failed to get user input:", err)
			return nil
		}

		switch answer {
		case "y", "Y":
			err = chat_(g.Factories["openai-chat"].(*chat.Step), contextManager)
			if err != nil {
				return err
			}
		case "n", "N":
			return nil
		}
	}

	return nil
}

func (g *GeppettoCommand) readStepResults(
	ctx context.Context,
	m *steps.StepResult[string],
	w io.Writer,
) (string, error) {
	accumulate := ""
	for {
		select {
		case <-ctx.Done():
			return accumulate, ctx.Err()
		case result, ok := <-m.GetChannel():
			if !ok {
				return accumulate, nil
			}

			if !result.Ok() {
				return accumulate, result.Error()
			}

			v, err := result.Value()
			if err != nil {
				return accumulate, err
			}

			_, err = w.Write([]byte(v))
			if err != nil {
				return accumulate, err
			}
			accumulate += v
		}
	}
}

func chat_(
	step *chat.Step,
	contextManager *geppetto_context.Manager,
) error {
	// switch on streaming for chatting
	step.StepSettings.Stream = true

	p := tea.NewProgram(
		ui.InitialModel(contextManager, step),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil

}

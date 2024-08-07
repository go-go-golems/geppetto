package cmds

import (
	"context"
	_ "embed"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	bobatea_chat "github.com/go-go-golems/bobatea/pkg/chat"
	"github.com/go-go-golems/bobatea/pkg/conversation"
	"github.com/go-go-golems/geppetto/pkg/events"
	"github.com/go-go-golems/geppetto/pkg/steps"
	"github.com/go-go-golems/geppetto/pkg/steps/ai"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/chat"
	"github.com/go-go-golems/geppetto/pkg/steps/ai/settings"
	"github.com/go-go-golems/geppetto/pkg/ui"
	glazedcmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/tcnksm/go-input"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"strings"
)

type GeppettoCommandDescription struct {
	Name      string                            `yaml:"name"`
	Short     string                            `yaml:"short"`
	Long      string                            `yaml:"long,omitempty"`
	Flags     []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
	Arguments []*parameters.ParameterDefinition `yaml:"arguments,omitempty"`
	Layers    []layers.ParameterLayer           `yaml:"layers,omitempty"`

	Prompt       string                  `yaml:"prompt,omitempty"`
	Messages     []*conversation.Message `yaml:"messages,omitempty"`
	SystemPrompt string                  `yaml:"system-prompt,omitempty"`
}

const GeppettoHelpersSlug = "geppetto-helpers"

func NewHelpersParameterLayer() (layers.ParameterLayer, error) {
	return layers.NewParameterLayer(GeppettoHelpersSlug, "Geppetto helpers",
		layers.WithParameterDefinitions(
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
			parameters.NewParameterDefinition(
				"interactive",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Ask for chat continuation after inference"),
				parameters.WithDefault(true),
			),
			parameters.NewParameterDefinition(
				"chat",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Automatically continue in chat mode"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition(
				"force-interactive",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Always enter interactive mode, even with non-tty stdout"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition(
				"images",
				parameters.ParameterTypeFileList,
				parameters.WithHelp("Images to display"),
			),
		),
	)
}

type HelpersSettings struct {
	PrintPrompt                 bool                   `glazed.parameter:"print-prompt"`
	System                      string                 `glazed.parameter:"system"`
	AppendMessageFile           string                 `glazed.parameter:"append-message-file"`
	MessageFile                 string                 `glazed.parameter:"message-file"`
	AutomaticallyContinueInChat bool                   `glazed.parameter:"chat"`
	Interactive                 bool                   `glazed.parameter:"interactive"`
	ForceInteractive            bool                   `glazed.parameter:"force-interactive"`
	Images                      []*parameters.FileData `glazed.parameter:"images"`
}

type GeppettoCommand struct {
	*glazedcmds.CommandDescription `yaml:",inline"`
	StepSettings                   *settings.StepSettings  `yaml:"stepSettings,omitempty"`
	Prompt                         string                  `yaml:"prompt,omitempty"`
	Messages                       []*conversation.Message `yaml:"messages,omitempty"`
	SystemPrompt                   string                  `yaml:"system-prompt,omitempty"`
}

var _ glazedcmds.WriterCommand = &GeppettoCommand{}

type GeppettoCommandOption func(*GeppettoCommand)

func WithPrompt(prompt string) GeppettoCommandOption {
	return func(g *GeppettoCommand) {
		g.Prompt = prompt
	}
}

func WithMessages(messages []*conversation.Message) GeppettoCommandOption {
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
	settings *settings.StepSettings,
	options ...GeppettoCommandOption,
) (*GeppettoCommand, error) {
	helpersParameterLayer, err := NewHelpersParameterLayer()
	if err != nil {
		return nil, err
	}

	description.Layers.PrependLayers(helpersParameterLayer)

	ret := &GeppettoCommand{
		CommandDescription: description,
		StepSettings:       settings,
	}

	for _, option := range options {
		option(ret)
	}

	return ret, nil
}

func (g *GeppettoCommand) InitializeContextManager(
	contextManager conversation.Manager,
	helperSettings *HelpersSettings,
	ps map[string]interface{},
) error {
	if g.SystemPrompt != "" {
		systemPromptTemplate, err := templating.CreateTemplate("system-prompt").Parse(g.SystemPrompt)
		if err != nil {
			return err
		}

		var systemPromptBuffer strings.Builder
		err = systemPromptTemplate.Execute(&systemPromptBuffer, ps)
		if err != nil {
			return err
		}

		// TODO(manuel, 2023-12-07) Only do this conditionally, or maybe if the system prompt hasn't been set yet, if you use an agent.
		contextManager.AppendMessages(conversation.NewChatMessage(
			conversation.RoleSystem,
			systemPromptBuffer.String(),
		))
	}

	for _, message_ := range g.Messages {
		switch content := message_.Content.(type) {
		case *conversation.ChatMessageContent:
			messageTemplate, err := templating.CreateTemplate("message").Parse(content.Text)
			if err != nil {
				return err
			}

			var messageBuffer strings.Builder
			err = messageTemplate.Execute(&messageBuffer, ps)
			if err != nil {
				return err
			}
			s_ := messageBuffer.String()

			contextManager.AppendMessages(conversation.NewChatMessage(
				content.Role, s_, conversation.WithTime(message_.Time)))
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

		images := []*conversation.ImageContent{}
		for _, img := range helperSettings.Images {
			image, err := conversation.NewImageContentFromFile(img.Path)
			if err != nil {
				return err
			}

			images = append(images, image)
		}
		initialPrompt := promptBuffer.String()
		messageContent := &conversation.ChatMessageContent{
			Role:   conversation.RoleUser,
			Text:   initialPrompt,
			Images: images,
		}
		contextManager.AppendMessages(conversation.NewMessage(messageContent))
	}

	return nil
}

func (g *GeppettoCommand) Run(
	ctx context.Context,
	step steps.Step[conversation.Conversation, string],
	contextManager conversation.Manager,
	helpersSettings *HelpersSettings,
	ps map[string]interface{},
) (steps.StepResult[string], error) {
	err := g.InitializeContextManager(contextManager, helpersSettings, ps)
	if err != nil {
		return nil, err
	}

	conversation_ := contextManager.GetConversation()
	if helpersSettings.PrintPrompt {
		fmt.Println(conversation_.GetSinglePrompt())
		return nil, nil
	}

	messagesM := steps.Resolve(conversation_)
	m := steps.Bind[conversation.Conversation, string](ctx, messagesM, step)

	return m, nil
}

// RunIntoWriter runs the command and writes the output into the given writer.
func (g *GeppettoCommand) RunIntoWriter(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	w io.Writer,
) error {
	if g.Prompt != "" && len(g.Messages) != 0 {
		return errors.Errorf("Prompt and messages are mutually exclusive")
	}

	s := &HelpersSettings{}
	err := parsedLayers.InitializeStruct(GeppettoHelpersSlug, s)
	if err != nil {
		return errors.Wrap(err, "failed to initialize settings")
	}

	endedInNewline := false

	stepSettings := g.StepSettings.Clone()

	err = stepSettings.UpdateFromParsedLayers(parsedLayers)
	if err != nil {
		return err
	}

	stepFactory := &ai.StandardStepFactory{
		Settings: stepSettings,
	}

	router, err := events.NewEventRouter()
	if err != nil {
		return err
	}

	defer func() {
		err := router.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close pubSub")
		}
	}()

	router.AddHandler("chat", "chat", chat.StepPrinterFunc("", w))

	contextManager := conversation.NewManager()

	var chatStep chat.Step
	chatStep, err = stepFactory.NewStep(chat.WithPublishedTopic(router.Publisher, "chat"))
	if err != nil {
		return err
	}

	// load and render the system prompt
	if s.System != "" {
		g.SystemPrompt = s.System
	}

	// load and render messages
	if s.MessageFile != "" {
		messages_, err := conversation.LoadFromFile(s.MessageFile)
		if err != nil {
			return err
		}

		g.Messages = messages_
	}

	if s.AppendMessageFile != "" {
		messages_, err := conversation.LoadFromFile(s.AppendMessageFile)
		if err != nil {
			return err
		}
		g.Messages = append(g.Messages, messages_...)
	}

	ctx, cancel := context.WithCancel(ctx)

	eg := errgroup.Group{}
	eg.Go(func() error {
		defer cancel()

		// TODO(manuel, 2024-04-26) We really should only pass the default slug here
		val, present := parsedLayers.Get(layers.DefaultSlug)
		if !present {
			return errors.New("could not get default layer")
		}
		m, err := g.Run(ctx, chatStep, contextManager, s, val.Parameters.ToMap())

		if err != nil {
			return err
		}
		if m == nil {
			return nil
		}

		isStream := stepSettings.Chat.Stream
		log.Debug().Bool("isStream", isStream).Msg("")

		res := m.Return()
		for _, msg := range res {
			s, err := msg.Value()
			if err != nil {
				// TODO(manuel, 2023-12-09) Better error handling here, to catch I guess streaming error and HTTP errors
				return err
			} else {
				contextManager.AppendMessages(conversation.NewChatMessage(conversation.RoleAssistant, s))

				if !isStream {
					_, err := w.Write([]byte(s))
					if err != nil {
						return err
					}
					endedInNewline = strings.HasSuffix(s, "\n")
				}
			}
		}

		// check if terminal is tty
		isOutputTerminal := isatty.IsTerminal(os.Stdout.Fd())
		forceInteractive := s.ForceInteractive
		continueInChat := s.AutomaticallyContinueInChat && s.Interactive
		askChat := (isOutputTerminal || forceInteractive) && !s.AutomaticallyContinueInChat && s.Interactive

		lengthBeforeChat := len(contextManager.GetConversation())

		if askChat {
			if !endedInNewline {
				fmt.Println()
			}

			continueInChat, err = g.askForChatContinuation(continueInChat)
			if err != nil {
				return err
			}
		}

		if continueInChat {
			stepFactory.Settings.Chat.Stream = true
			chatStep, err = stepFactory.NewStep(chat.WithPublishedTopic(router.Publisher, "ui"))
			if err != nil {
				return err
			}

			err = chat_(ctx, chatStep, router, contextManager)
			if err != nil {
				return err
			}

			fmt.Printf("\n---\n")
			for idx, msg := range contextManager.GetConversation() {
				if idx < lengthBeforeChat {
					continue
				}
				view := msg.Content.View()
				fmt.Printf("\n%s\n", view)
			}
			if err != nil {
				return err
			}
		}

		return nil
	})
	eg.Go(func() error {
		return router.Run(ctx)
	})

	return eg.Wait()
}

func (g *GeppettoCommand) askForChatContinuation(continueInChat bool) (bool, error) {
	tty_, err := bobatea_chat.OpenTTY()
	if err != nil {
		return false, err
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
				return errors.Errorf("please enter 'y' or 'n'")
			}
		},
	})

	if err != nil {
		fmt.Println("Failed to get user input:", err)
		return false, err
	}

	switch answer {
	case "y", "Y":
		continueInChat = true

	case "n", "N":
		return false, nil
	}
	return continueInChat, nil
}

func chat_(
	ctx context.Context,
	step chat.Step,
	router *events.EventRouter,
	contextManager conversation.Manager,
) error {
	// switch on streaming for chatting
	// TODO(manuel, 2023-12-09) Probably need to create a new follow on step for enabling streaming

	isOutputTerminal := isatty.IsTerminal(os.Stdout.Fd())

	options := []tea.ProgramOption{
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	}
	if !isOutputTerminal {
		options = append(options, tea.WithOutput(os.Stderr))
	} else {
		options = append(options, tea.WithAltScreen())
	}

	backend := ui.NewStepBackend(step)

	model := bobatea_chat.InitialModel(
		contextManager,
		backend,
		bobatea_chat.WithTitle("PINOCCHIO AT YOUR SERVICE:"),
	)

	p := tea.NewProgram(
		model,
		options...,
	)

	router.AddHandler("ui", "ui", ui.StepChatForwardFunc(p))
	err := router.RunHandlers(ctx)
	if err != nil {
		return err
	}

	if _, err = p.Run(); err != nil {
		return err
	}

	return nil

}

package builder

import (
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"strings"

	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/pkg/errors"
)

// ManagerBuilder helps construct a conversation.Manager with the given settings
type ManagerBuilder struct {
	systemPrompt string
	messages     []*conversation.Message
	prompt       string
	variables    map[string]interface{}
	images       []string

	autosaveEnabled  bool
	autosaveTemplate string
	autosavePath     string
}

type ConversationManagerOption func(*ManagerBuilder) error

func WithSystemPrompt(systemPrompt string) ConversationManagerOption {
	return func(b *ManagerBuilder) error {
		b.systemPrompt = systemPrompt
		return nil
	}
}

func WithMessages(messages []*conversation.Message) ConversationManagerOption {
	return func(b *ManagerBuilder) error {
		b.messages = messages
		return nil
	}
}

func WithPrompt(prompt string) ConversationManagerOption {
	return func(b *ManagerBuilder) error {
		b.prompt = prompt
		return nil
	}
}

func WithVariables(variables map[string]interface{}) ConversationManagerOption {
	return func(b *ManagerBuilder) error {
		b.variables = variables
		return nil
	}
}

func WithImages(images []string) ConversationManagerOption {
	return func(b *ManagerBuilder) error {
		b.images = images
		return nil
	}
}

type AutosaveSettings struct {
	Enabled  bool
	Template string
	Path     string
}

func WithAutosaveSettings(settings AutosaveSettings) ConversationManagerOption {
	return func(b *ManagerBuilder) error {
		b.autosaveEnabled = settings.Enabled
		b.autosaveTemplate = settings.Template
		b.autosavePath = settings.Path
		return nil
	}
}

// NewConversationManagerBuilder creates a new builder for conversation.Manager
func NewConversationManagerBuilder(options ...ConversationManagerOption) (*ManagerBuilder, error) {
	builder := &ManagerBuilder{
		variables: make(map[string]interface{}),
	}

	for _, opt := range options {
		if err := opt(builder); err != nil {
			return nil, err
		}
	}

	return builder, nil
}

// Build creates and initializes a new conversation.Manager
func (b *ManagerBuilder) Build() (conversation.Manager, error) {
	enabled := "no"
	if b.autosaveEnabled {
		enabled = "yes"
	}

	manager := conversation.NewManager(
		conversation.WithAutosave(
			enabled,
			b.autosaveTemplate,
			b.autosavePath,
		),
	)

	if err := b.initializeConversation(manager); err != nil {
		return nil, err
	}

	return manager, nil
}

func (b *ManagerBuilder) initializeConversation(manager conversation.Manager) error {
	if b.systemPrompt != "" {
		systemPromptTemplate, err := templating.CreateTemplate("system-prompt").Parse(b.systemPrompt)
		if err != nil {
			return errors.Wrap(err, "failed to parse system prompt template")
		}

		var systemPromptBuffer strings.Builder
		err = systemPromptTemplate.Execute(&systemPromptBuffer, b.variables)
		if err != nil {
			return errors.Wrap(err, "failed to execute system prompt template")
		}

		manager.AppendMessages(conversation.NewChatMessage(
			conversation.RoleSystem,
			systemPromptBuffer.String(),
		))
	}

	for _, message_ := range b.messages {
		switch content := message_.Content.(type) {
		case *conversation.ChatMessageContent:
			messageTemplate, err := templating.CreateTemplate("message").Parse(content.Text)
			if err != nil {
				return errors.Wrap(err, "failed to parse message template")
			}

			var messageBuffer strings.Builder
			err = messageTemplate.Execute(&messageBuffer, b.variables)
			if err != nil {
				return errors.Wrap(err, "failed to execute message template")
			}
			s_ := messageBuffer.String()

			manager.AppendMessages(conversation.NewChatMessage(
				content.Role, s_, conversation.WithTime(message_.Time)))
		}
	}

	if b.prompt != "" {
		promptTemplate, err := templating.CreateTemplate("prompt").Parse(b.prompt)
		if err != nil {
			return errors.Wrap(err, "failed to parse prompt template")
		}

		var promptBuffer strings.Builder
		err = promptTemplate.Execute(&promptBuffer, b.variables)
		if err != nil {
			return errors.Wrap(err, "failed to execute prompt template")
		}

		images := []*conversation.ImageContent{}
		for _, imagePath := range b.images {
			image, err := conversation.NewImageContentFromFile(imagePath)
			if err != nil {
				return errors.Wrap(err, "failed to create image content")
			}
			images = append(images, image)
		}

		messageContent := &conversation.ChatMessageContent{
			Role:   conversation.RoleUser,
			Text:   promptBuffer.String(),
			Images: images,
		}
		manager.AppendMessages(conversation.NewMessage(messageContent))
	}

	return nil
}

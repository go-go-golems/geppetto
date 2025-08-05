package builder

import (
	"github.com/go-go-golems/geppetto/pkg/conversation"
	"github.com/pkg/errors"
	"strings"

	"github.com/go-go-golems/glazed/pkg/helpers/templating"
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
// NewManagerBuilder creates a new builder for conversation.Manager
func NewManagerBuilder() *ManagerBuilder {
	return &ManagerBuilder{
		variables: make(map[string]interface{}),
	}
}

func (b *ManagerBuilder) WithSystemPrompt(systemPrompt string) *ManagerBuilder {
	b.systemPrompt = systemPrompt
	return b
}

func (b *ManagerBuilder) WithMessages(messages []*conversation.Message) *ManagerBuilder {
	b.messages = messages
	return b
}

func (b *ManagerBuilder) WithPrompt(prompt string) *ManagerBuilder {
	b.prompt = prompt
	return b
}

func (b *ManagerBuilder) WithVariables(variables map[string]interface{}) *ManagerBuilder {
	if b.variables == nil {
		b.variables = make(map[string]interface{})
	}
	for k, v := range variables {
		b.variables[k] = v
	}
	return b
}

func (b *ManagerBuilder) WithImages(images []string) *ManagerBuilder {
	b.images = images
	return b
}

type AutosaveSettings struct {
	Enabled  bool
	Template string
	Path     string
}

func (b *ManagerBuilder) WithAutosaveSettings(settings AutosaveSettings) *ManagerBuilder {
	b.autosaveEnabled = settings.Enabled
	b.autosaveTemplate = settings.Template
	b.autosavePath = settings.Path
	return b
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

		if err := manager.AppendMessages(conversation.NewChatMessage(
			conversation.RoleSystem,
			systemPromptBuffer.String(),
		)); err != nil {
			return errors.Wrap(err, "failed to append system prompt message")
		}
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

			if err := manager.AppendMessages(conversation.NewChatMessage(
				content.Role, s_, conversation.WithTime(message_.Time))); err != nil {
				return errors.Wrap(err, "failed to append template-rendered message")
			}
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
		if err := manager.AppendMessages(conversation.NewMessage(messageContent)); err != nil {
			return errors.Wrap(err, "failed to append prompt message with images")
		}
	}

	return nil
}

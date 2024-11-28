package conversation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/helpers/maps"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/google/uuid"
)

type ManagerImpl struct {
	Tree            *ConversationTree
	ConversationID  uuid.UUID
	autosaveEnabled bool
	autosaveFormat  string
	autosaveDir     string
}

var _ Manager = (*ManagerImpl)(nil)

type ManagerOption func(*ManagerImpl)

func WithMessages(messages ...*Message) ManagerOption {
	return func(m *ManagerImpl) {
		m.AppendMessages(messages...)
	}
}

func WithManagerConversationID(conversationID uuid.UUID) ManagerOption {
	return func(m *ManagerImpl) {
		m.ConversationID = conversationID
	}
}

func WithAutosave(enabled bool, format string, dir string) ManagerOption {
	return func(m *ManagerImpl) {
		m.autosaveEnabled = enabled
		m.autosaveDir = dir
		if format == "" {
			m.autosaveFormat = "{{.Year}}/{{.Month}}/{{.Day}}/{{.ConversationID}}-{{.Time.Format \"150405\"}}-{{len .Messages}}.json"
		} else {
			m.autosaveFormat = format
		}
	}
}

// CreateManager initializes a Manager implementation with system prompts, initial messages,
// and customizable options. It handles template rendering for prompts and messages.
//
// NOTE(manuel, 2024-04-07) This currently seems to only be used by the codegen tests,
// while the main geppetto command uses NewManager. Unclear if this is just a legacy helper.
//
// The systemPrompt and prompt templates are rendered using the params.
// Messages are also rendered using the params before being added to the manager.
//
// ManagerOptions can be passed to further customize the manager on creation.
func CreateManager(
	systemPrompt string,
	prompt string,
	messages []*Message,
	params interface{},
	options ...ManagerOption,
) (*ManagerImpl, error) {
	// convert the params to map[string]interface{}
	var ps map[string]interface{}
	if _, ok := params.(map[string]interface{}); !ok {
		var err error
		ps, err = maps.GlazedStructToMap(params)
		if err != nil {
			return nil, err
		}
	} else {
		ps = params.(map[string]interface{})
	}

	manager := NewManager()

	if systemPrompt != "" {
		systemPromptTemplate, err := templating.CreateTemplate("system-prompt").Parse(systemPrompt)
		if err != nil {
			return nil, err
		}

		var systemPromptBuffer strings.Builder
		err = systemPromptTemplate.Execute(&systemPromptBuffer, ps)
		if err != nil {
			return nil, err
		}

		// TODO(manuel, 2023-12-07) Only do this conditionally, or maybe if the system prompt hasn't been set yet, if you use an agent.
		manager.AppendMessages(NewChatMessage(RoleSystem, systemPromptBuffer.String()))
	}

	for _, message := range messages {
		if msg, ok := message.Content.(*ChatMessageContent); ok {
			messageTemplate, err := templating.CreateTemplate("message").Parse(msg.Text)
			if err != nil {
				return nil, err
			}

			var messageBuffer strings.Builder
			err = messageTemplate.Execute(&messageBuffer, ps)
			if err != nil {
				return nil, err
			}
			s_ := messageBuffer.String()

			manager.AppendMessages(NewChatMessage(msg.Role, s_, WithTime(message.Time)))
		}
	}

	// render the prompt
	if prompt != "" {
		// TODO(manuel, 2023-02-04) All this could be handle by some prompt renderer kind of thing
		promptTemplate, err := templating.CreateTemplate("prompt").Parse(prompt)
		if err != nil {
			return nil, err
		}

		// TODO(manuel, 2023-02-04) This is where multisteps would work differently, since
		// the prompt would be rendered at execution time
		var promptBuffer strings.Builder
		err = promptTemplate.Execute(&promptBuffer, ps)
		if err != nil {
			return nil, err
		}

		manager.AppendMessages(NewChatMessage(RoleUser, promptBuffer.String()))
	}

	for _, option := range options {
		option(manager)
	}

	return manager, nil
}

func NewManager(options ...ManagerOption) *ManagerImpl {
	ret := &ManagerImpl{
		ConversationID: uuid.Nil,
		Tree:           NewConversationTree(),
	}
	for _, option := range options {
		option(ret)
	}

	if ret.ConversationID == uuid.Nil {
		ret.ConversationID = uuid.New()
	}

	return ret
}

func (c *ManagerImpl) GetConversation() Conversation {
	return c.Tree.GetLeftMostThread(c.Tree.RootID)
}

func (c *ManagerImpl) GetMessage(ID NodeID) (*Message, bool) {
	return c.Tree.GetMessageByID(ID)
}

func (c *ManagerImpl) AppendMessages(messages ...*Message) {
	c.Tree.AppendMessages(messages)
	if c.autosaveEnabled {
		_ = c.autoSave() // Intentionally ignoring errors for now
	}
}

func (c *ManagerImpl) AttachMessages(parentID NodeID, messages ...*Message) {
	c.Tree.AttachThread(parentID, messages)
}

func (c *ManagerImpl) PrependMessages(messages ...*Message) {
	c.Tree.PrependThread(messages)
}

// SaveToFile persists the current conversation state to a JSON file, enabling
// conversation continuity across sessions.
func (c *ManagerImpl) SaveToFile(s string) error {
	// TODO(manuel, 2023-11-14) For now only json
	msgs := c.GetConversation()
	f, err := os.Create(s)
	if err != nil {
		return err
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	// TODO(manuel, 2024-04-07) Encode as tree structure?? we skip the Children field on purpose to avoid circular references
	err = encoder.Encode(msgs)
	if err != nil {
		return err
	}

	return nil
}

func (c *ManagerImpl) autoSave() error {
	now := time.Now()
	data := map[string]interface{}{
		"Year":           now.Format("2006"),
		"Month":          now.Format("01"),
		"Day":            now.Format("02"),
		"ConversationID": c.ConversationID.String(),
		"Messages":       c.GetConversation(),
		"Tree":           c.Tree,
		"Time":           now,
	}

	tmpl, err := templating.CreateTemplate("autosave").Parse(c.autosaveFormat)
	if err != nil {
		return err
	}

	var filePathBuffer strings.Builder
	err = tmpl.Execute(&filePathBuffer, data)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(c.autosaveDir, filePathBuffer.String())

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return c.SaveToFile(fullPath)
}

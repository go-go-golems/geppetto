package conversation

import (
	"github.com/go-go-golems/glazed/pkg/helpers/maps"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/google/uuid"
	"strings"
)

// Manager defines the interface for high-level conversation management operations.
type Manager interface {
	GetConversation() Conversation
	AppendMessages(msgs ...*Message)
	AttachMessages(parentID NodeID, msgs ...*Message)
	GetMessage(ID NodeID) (*Message, bool)
	SaveToFile(filename string) error
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

type ManagerImpl struct {
	Tree           *ConversationTree
	ConversationID uuid.UUID
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

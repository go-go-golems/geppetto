package conversation

import (
	"encoding/json"
	"github.com/google/uuid"
	"os"
)

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

func (c *ManagerImpl) AppendMessages(messages ...*Message) {
	c.Tree.AppendMessages(messages)
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

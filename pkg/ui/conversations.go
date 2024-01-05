package ui

import (
	"github.com/go-go-golems/bobatea/pkg/chat"
	"github.com/go-go-golems/geppetto/pkg/context"
)

type GeppettoConversationManager struct {
	manager *context.Manager
}

func NewGeppettoConversationManager(manager *context.Manager) *GeppettoConversationManager {
	return &GeppettoConversationManager{
		manager: manager,
	}
}

func (g *GeppettoConversationManager) GetMessages() []*chat.Message {
	msgs := g.manager.GetMessages()
	ret := make([]*chat.Message, len(msgs))
	for i, m := range msgs {

		ret[i] = &chat.Message{
			Text:           m.Text,
			Time:           m.Time,
			Role:           m.Role,
			ID:             m.ID,
			ParentID:       m.ParentID,
			ConversationID: m.ConversationID,
			Metadata:       m.Metadata,
		}
	}

	return ret
}

func (g *GeppettoConversationManager) GetMessagesWithSystemPrompt() []*chat.Message {
	msgs := g.manager.GetMessagesWithSystemPrompt()
	ret := make([]*chat.Message, len(msgs))
	for i, m := range msgs {

		ret[i] = &chat.Message{
			Text:           m.Text,
			Time:           m.Time,
			Role:           m.Role,
			ID:             m.ID,
			ParentID:       m.ParentID,
			ConversationID: m.ConversationID,
			Metadata:       m.Metadata,
		}
	}
	return ret
}

func (g *GeppettoConversationManager) AddMessages(msgs ...*chat.Message) {
	msgs_ := make([]*context.Message, len(msgs))
	for i, m := range msgs {
		msgs_[i] = &context.Message{
			Text:           m.Text,
			Time:           m.Time,
			Role:           m.Role,
			ID:             m.ID,
			ParentID:       m.ParentID,
			ConversationID: m.ConversationID,
			Metadata:       m.Metadata,
		}
	}
	g.manager.AddMessages(msgs_...)
}

func (g *GeppettoConversationManager) SaveToFile(filename string) error {
	return g.manager.SaveToFile(filename)
}

var _ chat.ConversationManager = &GeppettoConversationManager{}

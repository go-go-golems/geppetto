package conversation

// Package conversation provides functionality for managing AI chat conversations.
//
// The conversation package implements a tree-based conversation structure that can handle
// both linear and branching chat histories. It supports different message roles (system, user, assistant),
// conversation persistence, and template-based message rendering.
//
// The Manager interface provides the main entry point for conversation operations:
// - Creating and managing conversation threads
// - Appending and attaching messages
// - Retrieving messages and conversation history
// - Saving conversations to persistent storage
//
// The package uses a tree data structure to represent conversations, allowing for
// features like conversation branching, message threading, and maintaining the full
// history of interactions while providing easy access to the current conversation thread.

// Manager defines the interface for high-level conversation management operations.
type Manager interface {
	GetConversation() Conversation
	AppendMessages(msgs ...*Message)
	AttachMessages(parentID NodeID, msgs ...*Message)
	GetMessage(ID NodeID) (*Message, bool)
	SaveToFile(filename string) error
}

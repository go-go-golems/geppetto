# Conversation Package

The `conversation` package provides a flexible structure for managing complex conversation flows in LLM chatbots. It offers a tree-like structure for storing and traversing conversation messages.

## Key Components

1. **Message**: Represents individual messages with various content types.
2. **ConversationTree**: Manages the tree structure of the conversation.
3. **Manager**: Provides high-level conversation management operations.
4. **Context**: Handles loading and saving conversations from/to files.

## Features

- Tree-based conversation representation
- Support for different message content types (chat, tool use, tool results, images)
- Flexible conversation traversal methods
- JSON/YAML file persistence
- High-level conversation management interface

## Usage Examples

### Creating and Manipulating a Conversation Tree

This example demonstrates how to create a conversation tree, add messages to it, and retrieve conversation threads. It showcases the basic operations for building and traversing the conversation structure.

```go
tree := conversation.NewConversationTree()

message1 := conversation.NewChatMessage(conversation.RoleUser, "Hello!")
message2 := conversation.NewChatMessage(conversation.RoleAssistant, "Hi there!")

tree.InsertMessages(message1, message2)

thread := tree.GetConversationThread(message2.ID)
leftmostThread := tree.GetLeftMostThread(tree.RootID)
```

### Using the Manager

The Manager provides a higher-level interface for managing conversations. This example shows how to create a manager with initial prompts, add messages, and retrieve the conversation. It's useful for more complex conversation handling scenarios.

```go
manager, err := conversation.CreateManager(
    "System prompt",
    "User prompt",
    []*conversation.Message{},
    nil,
)
if err != nil {
    // Handle error
}

manager.AppendMessages(message1, message2)
conversation := manager.GetConversation()
```

### Persistence

This example illustrates how to save and load conversation trees to/from JSON files. This feature is crucial for maintaining conversation state across sessions or for analysis purposes.

```go
err := tree.SaveToFile("conversation.json")
if err != nil {
    // Handle error
}

loadedTree := conversation.NewConversationTree()
err = loadedTree.LoadFromFile("conversation.json")
if err != nil {
    // Handle error
}
```

## Extending Message Content Types

The package allows for custom message content types by implementing the MessageContent interface. This flexibility enables the conversation package to handle various types of interactions beyond simple text messages.

```go
type MessageContent interface {
    ContentType() ContentType
    String() string
    View() string
}
```

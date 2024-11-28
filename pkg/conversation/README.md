# Conversation Package

The `conversation` package provides a sophisticated system for managing complex conversation flows in LLM chatbots. It implements a tree-based conversation structure that supports branching dialogues, message threading, and various types of content.

## Core Concepts

### Message Structure
Messages are the fundamental units of conversation. Each message has:
- **ID**: A unique identifier for the message
- **Role**: The participant role (system, user, assistant, tool)
- **Content**: The actual content of the message (chat text, tool calls, results, etc.)
- **Time**: Timestamp of the message
- **Metadata**: Additional information about the message

### Tree-Based Conversation Model
Conversations are structured as trees, allowing for:
- **Linear Conversations**: Simple back-and-forth dialogues
- **Branching Dialogues**: Multiple conversation paths from a single point
- **Message Threading**: Linking related messages in a conversation
- **Context Preservation**: Maintaining conversation history and state

### Message Content Types
The package supports multiple content types:
1. **ChatMessageContent**: Standard text messages
2. **ToolCallContent**: Requests for tool operations
3. **ToolResultContent**: Results from tool executions
4. **ImageContent**: Image data and metadata
5. **Custom Content**: Extensible for additional content types

## Key Components

### 1. ConversationTree
The core data structure managing message relationships:
```go
type ConversationTree struct {
    Root   NodeID
    Nodes  map[NodeID]*Node
    Edges  map[NodeID][]NodeID
}
```

Key operations:
- `InsertMessages`: Add new messages to the tree
- `GetMessageByID`: Retrieve specific messages
- `GetConversationThread`: Get a sequence of related messages
- `GetLeftMostThread`: Get the primary conversation path

### 2. Manager
High-level interface for conversation operations:
```go
type Manager interface {
    GetConversation() Conversation
    AppendMessages(msgs ...*Message)
    AttachMessages(parentID NodeID, msgs ...*Message)
    GetMessage(ID NodeID) (*Message, bool)
    SaveToFile(filename string) error
}
```

Features:
- Message management and organization
- Conversation state tracking
- Template-based prompt handling
- Persistence operations

### 3. Context
Handles conversation persistence and loading:
```go
type Context interface {
    LoadFromFile(filename string) error
    SaveToFile(filename string) error
    GetConversation() Conversation
}
```

## Implementation Guide

### 1. Setting Up a Conversation Manager
```go
// Initialize with system prompt and configuration
manager := conversation.NewManager(
    conversation.WithManagerConversationID(uuid.New()),
)

// Add system prompt
manager.AppendMessages(conversation.NewChatMessage(
    conversation.RoleSystem,
    "System initialization prompt",
))
```

### 2. Handling User Messages
```go
// Add user message
userMsg := conversation.NewChatMessage(
    conversation.RoleUser,
    "User input text",
)
manager.AppendMessages(userMsg)

// Get the current conversation state
conv := manager.GetConversation()
```

### 3. Managing Assistant Responses
```go
// Add assistant response
response := conversation.NewChatMessage(
    conversation.RoleAssistant,
    "Assistant response",
)
manager.AppendMessages(response)

// Add tool calls if needed
toolCall := conversation.NewToolCallMessage(
    "tool_name",
    map[string]interface{}{"param": "value"},
)
manager.AppendMessages(toolCall)
```

### 4. Branching Conversations
```go
// Create a new branch from an existing message
parentID := existingMessage.ID
newBranch := conversation.NewChatMessage(
    conversation.RoleAssistant,
    "Alternative response",
)
manager.AttachMessages(parentID, newBranch)
```

### 5. Persistence
```go
// Save conversation state
err := manager.SaveToFile("conversation.json")
if err != nil {
    log.Fatal(err)
}

// Load existing conversation
loadedManager := conversation.NewManager()
err = loadedManager.LoadFromFile("conversation.json")
```

## Best Practices

1. **Message Organization**
   - Use appropriate roles for messages
   - Maintain clear parent-child relationships
   - Include relevant metadata

2. **State Management**
   - Regularly save conversation state
   - Handle branching conversations carefully
   - Track conversation context

3. **Error Handling**
   - Validate message content and structure
   - Handle loading/saving errors gracefully
   - Maintain conversation integrity

4. **Performance Considerations**
   - Limit conversation tree depth
   - Implement cleanup for old branches
   - Optimize message content size

## Extensions and Customization

### Custom Content Types
Implement the MessageContent interface for new content types:
```go
type CustomContent struct {
    Type    string
    Payload interface{}
}

func (c *CustomContent) ContentType() ContentType {
    return ContentTypeCustom
}

func (c *CustomContent) String() string {
    return fmt.Sprintf("%v", c.Payload)
}
```

### Custom Manager Options
Add specialized behavior with manager options:
```go
func WithCustomBehavior(config interface{}) conversation.ManagerOption {
    return func(m *conversation.ManagerImpl) {
        // Implement custom behavior
    }
}
```

## Error Handling

Common errors and their handling:
1. **Invalid Message Structure**
   ```go
   if !message.IsValid() {
       return fmt.Errorf("invalid message structure: %v", message)
   }
   ```

2. **Tree Manipulation Errors**
   ```go
   if _, exists := tree.GetMessageByID(parentID); !exists {
       return fmt.Errorf("parent message not found: %v", parentID)
   }
   ```

3. **Persistence Errors**
   ```go
   if err := manager.SaveToFile(filename); err != nil {
       log.Printf("Failed to save conversation: %v", err)
       // Implement recovery strategy
   }
   ```

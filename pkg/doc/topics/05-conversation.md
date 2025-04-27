---
Title: Understanding and Using the Conversation Package
Slug: geppetto-conversation-package
Short: A guide to the conversation package in Geppetto, covering message types, conversation trees, and metadata management.
Topics:
- geppetto
- conversation
- architecture
- tutorial
- messages
Commands: []
Flags: []
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Tutorial
---

# Understanding and Using the Conversation Package

This tutorial provides a comprehensive guide to using the `conversation` package in Geppetto, which is responsible for managing structured conversations between users and AI assistants. The package provides a rich set of types and functions for representing different kinds of message content, organizing messages into conversation trees, and handling metadata.

## Core Concepts

The conversation package is built around several key types:

- `Message`: Represents a single message in a conversation with content, metadata, and tree relationships
- `MessageContent`: Interface for different types of content (text, images, tool calls, etc.)
- `Conversation`: A slice of Message pointers representing a sequence of messages
- `Role`: Identifies the sender of a message (user, assistant, system, tool)
- `ContentType`: Identifies the type of content in a message

### Message Content Types

The package supports several types of message content:

- `ChatMessageContent`: Regular text messages with a role (user, assistant, system)
- `ToolUseContent`: Tool call requests made by the assistant
- `ToolResultContent`: Results returned from tool executions
- `ImageContent`: Images included in messages

## Creating Messages

### Basic Chat Messages

The simplest way to create a new message is using the `NewChatMessage` function:

```go
package main

import (
    "fmt"
    "github.com/your-org/geppetto/pkg/conversation"
)

func main() {
    // Create a simple user message
    userMsg := conversation.NewChatMessage(conversation.RoleUser, "Hello, assistant!")
    
    // Create a simple assistant response
    assistantMsg := conversation.NewChatMessage(conversation.RoleAssistant, "Hello! How can I help you today?")
    
    fmt.Println(userMsg.Content.String())
    fmt.Println(assistantMsg.Content.String())
}
```

### Using Message Options

You can use option functions to customize messages:

```go
import (
    "time"
    "github.com/google/uuid"
    "github.com/your-org/geppetto/pkg/conversation"
)

func createCustomMessage() *conversation.Message {
    // Create a message with custom options
    msg := conversation.NewChatMessage(
        conversation.RoleUser,
        "Can you help me with my code?",
        conversation.WithParentID(conversation.NodeID(uuid.New())),
        conversation.WithTime(time.Now().Add(-5 * time.Minute)),
        conversation.WithMetadata(map[string]interface{}{
            "source": "web",
            "browser": "chrome",
        }),
    )
    
    return msg
}
```

### Creating Messages with Images

```go
func createMessageWithImage() (*conversation.Message, error) {
    // Create an image content from a local file
    imageContent, err := conversation.NewImageContentFromFile("path/to/image.png")
    if err != nil {
        return nil, err
    }
    
    // Create a chat message content with text and the image
    content := conversation.NewChatMessageContent(
        conversation.RoleUser,
        "Here's the screenshot of the error I'm seeing:",
        []*conversation.ImageContent{imageContent},
    )
    
    // Create a message with this complex content
    msg := conversation.NewChatMessageFromContent(content)
    
    return msg, nil
}
```

### Tool Use and Tool Results

```go
import (
    "encoding/json"
    "github.com/your-org/geppetto/pkg/conversation"
)

func createToolMessages() ([]*conversation.Message, error) {
    // Create a tool use message (assistant calling a tool)
    toolInputData := map[string]interface{}{
        "query": "golang timezone conversion",
        "limit": 5,
    }
    
    toolInputJSON, err := json.Marshal(toolInputData)
    if err != nil {
        return nil, err
    }
    
    toolUseContent := &conversation.ToolUseContent{
        ToolID: "web-search",
        Name:   "search",
        Input:  toolInputJSON,
        Type:   "function",
    }
    
    toolUseMsg := conversation.NewMessage(toolUseContent)
    
    // Create a tool result message
    toolResultContent := &conversation.ToolResultContent{
        ToolID: "web-search",
        Result: "Found 5 results for golang timezone conversion: [list of results]",
    }
    
    toolResultMsg := conversation.NewMessage(toolResultContent)
    
    return []*conversation.Message{toolUseMsg, toolResultMsg}, nil
}
```

## Building Conversations

You can create a conversation by combining multiple messages:

```go
func buildConversation() conversation.Conversation {
    // Create system message
    systemMsg := conversation.NewChatMessage(
        conversation.RoleSystem,
        "You are a helpful coding assistant specializing in Go.",
    )
    
    // Create user message
    userMsg := conversation.NewChatMessage(
        conversation.RoleUser,
        "How do I handle JSON in Go?",
    )
    
    // Create assistant message
    assistantMsg := conversation.NewChatMessage(
        conversation.RoleAssistant,
        "In Go, you can use the encoding/json package to handle JSON data...",
    )
    
    // Create conversation from messages
    conv := conversation.NewConversation(systemMsg, userMsg, assistantMsg)
    
    return conv
}
```

## Working with Message Trees

The conversation package supports tree-structured conversations with parents and children:

```go
func buildConversationTree() *conversation.Message {
    // Create root message
    rootMsg := conversation.NewChatMessage(
        conversation.RoleSystem,
        "You are a helpful assistant.",
    )
    
    // Create child message with parent ID
    childMsg := conversation.NewChatMessage(
        conversation.RoleUser,
        "Hello!",
        conversation.WithParentID(rootMsg.ID),
    )
    
    // Add child to parent's children list
    rootMsg.Children = append(rootMsg.Children, childMsg)
    
    // Create a response to the user
    responseMsg := conversation.NewChatMessage(
        conversation.RoleAssistant,
        "Hi there! How can I help you today?",
        conversation.WithParentID(childMsg.ID),
    )
    
    // Add response to user message's children
    childMsg.Children = append(childMsg.Children, responseMsg)
    
    return rootMsg
}
```

## Using Conversation Utility Methods

The Conversation type provides several utility methods:

```go
func demonstrateConversationMethods() {
    // Build a conversation
    conv := buildConversation() // Using the function from earlier
    
    // Get a single prompt string representation
    prompt := conv.GetSinglePrompt()
    fmt.Println("Single prompt:")
    fmt.Println(prompt)
    
    // Get a string representation of the conversation
    convString := conv.ToString()
    fmt.Println("\nConversation string:")
    fmt.Println(convString)
    
    // Get a hash of the conversation for caching
    hash := conv.HashBytes()
    fmt.Printf("\nConversation hash: %x\n", hash)
}
```

## Working with LLM Message Metadata

The Message type includes fields for tracking LLM-specific metadata:

```go
func addLLMMetadata(msg *conversation.Message) {
    // Set temperature value
    temperature := 0.7
    
    // Set max tokens
    maxTokens := 1024
    
    // Set stop reason
    stopReason := "stop"
    
    // Create usage stats
    usage := &conversation.Usage{
        InputTokens:  150,
        OutputTokens: 320,
    }
    
    // Create and assign LLM metadata
    llmMetadata := &conversation.LLMMessageMetadata{
        Engine:      "gpt-4",
        Temperature: &temperature,
        MaxTokens:   &maxTokens,
        StopReason:  &stopReason,
        Usage:       usage,
    }
    
    msg.LLMMessageMetadata = llmMetadata
}
```

## Practical Example: Building a Complete Conversation Flow

Here's a more complete example showing how to build and manipulate a conversation:

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/your-org/geppetto/pkg/conversation"
)

func main() {
    // Create a new conversation starting with a system message
    systemMsg := conversation.NewChatMessage(
        conversation.RoleSystem,
        "You are a helpful coding assistant specializing in Go.",
    )
    
    // Add a user question
    userMsg := conversation.NewChatMessage(
        conversation.RoleUser,
        "How do I parse JSON in Go?",
        conversation.WithParentID(systemMsg.ID),
    )
    
    // Add user message as child of system message
    systemMsg.Children = append(systemMsg.Children, userMsg)
    
    // Create the assistant's response
    assistantMsg := conversation.NewChatMessage(
        conversation.RoleAssistant,
        "To parse JSON in Go, you can use the encoding/json package...",
        conversation.WithParentID(userMsg.ID),
    )
    
    // Add assistant message as child of user message
    userMsg.Children = append(userMsg.Children, assistantMsg)
    
    // Create a follow-up question from the user
    followUpMsg := conversation.NewChatMessage(
        conversation.RoleUser,
        "What about unmarshaling into a struct?",
        conversation.WithParentID(assistantMsg.ID),
    )
    
    // Add follow-up as child of assistant message
    assistantMsg.Children = append(assistantMsg.Children, followUpMsg)
    
    // Create a linear conversation from this tree for API use
    // This walks from root to a specific leaf, following a path
    linearConversation := conversation.NewConversation(
        systemMsg,
        userMsg,
        assistantMsg,
        followUpMsg,
    )
    
    // Print the linear conversation
    fmt.Println("Linear conversation:")
    fmt.Println(linearConversation.GetSinglePrompt())
    
    // Add LLM metadata to assistant message
    addLLMMetadata(assistantMsg)
    
    // Display token usage stats
    if assistantMsg.LLMMessageMetadata != nil && assistantMsg.LLMMessageMetadata.Usage != nil {
        usage := assistantMsg.LLMMessageMetadata.Usage
        fmt.Printf("\nToken usage - Input: %d, Output: %d, Total: %d\n",
            usage.InputTokens,
            usage.OutputTokens,
            usage.InputTokens + usage.OutputTokens,
        )
    }
}
```

## Best Practices

When working with the conversation package, keep these best practices in mind:

- **Message IDs**: Always use proper IDs for messages to maintain correct tree relationships.
- **Parent-Child Relationships**: When building a conversation tree, ensure both the ParentID field is set and the message is added to its parent's Children slice.
- **Content Types**: Choose the appropriate content type for each message (chat message, tool use, tool result, image).
- **Metadata**: Use metadata fields to store additional information that might be useful for your application.
- **Tree Traversal**: When working with conversation trees, implement proper traversal algorithms to extract specific paths or subtrees.
- **Error Handling**: Always handle errors when working with functions that can return them, especially when dealing with images or JSON operations.

## Conclusion

The conversation package provides a flexible and powerful way to represent structured conversations between users and AI assistants. By understanding its core types and functions, you can build sophisticated conversation management systems that support various content types, metadata tracking, and tree-structured conversations. 
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

## Managing Conversations with the Manager

While you can build conversations manually by linking messages as shown in the "Building Conversations" and "Working with Message Trees" sections, the `conversation` package provides a `Manager` type (specifically `ManagerImpl`) to simplify this process, especially for typical LLM interaction flows. The `Manager` handles message organization, tree management, and optionally, automatic saving of conversations.

### Creating a Manager

You create a new conversation manager using the `NewManager` function. You can provide options to customize its behavior, such as enabling autosave.

```go
package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/your-org/geppetto/pkg/conversation"
)

func main() {
	// Create a basic manager
	manager := conversation.NewManager()
	fmt.Printf("Created manager with conversation ID: %s\n", manager.ConversationID)

	// Create a manager with autosave enabled (saves to ~/.pinocchio/history by default)
	managerWithAutosave := conversation.NewManager(
		conversation.WithAutosave("yes", "", ""), // enable, default format, default dir
	)
	fmt.Printf("Created manager with autosave enabled, ID: %s\n", managerWithAutosave.ConversationID)
	
	// Create a manager with a specific ID
	specificID := uuid.New()
	managerWithID := conversation.NewManager(
		conversation.WithManagerConversationID(specificID),
	)
	fmt.Printf("Created manager with specific ID: %s\n", managerWithID.ConversationID)

}

```

### Adding Messages

The `Manager` automatically handles adding messages to the conversation tree. The `AppendMessages` method adds messages sequentially, extending the main conversation thread.

```go
func addMessagesToManager() {
	manager := conversation.NewManager()

	// Add a system message
	systemMsg := conversation.NewChatMessage(conversation.RoleSystem, "You are a Go expert.")
	manager.AppendMessages(systemMsg)

	// Add a user message (implicitly becomes a child of the system message)
	userMsg := conversation.NewChatMessage(conversation.RoleUser, "How do I use channels?")
	manager.AppendMessages(userMsg)

	// Add an assistant response (implicitly becomes a child of the user message)
	assistantMsg := conversation.NewChatMessage(conversation.RoleAssistant, "Channels are used for communication between goroutines...")
	manager.AppendMessages(assistantMsg)

	// If autosave is enabled, these messages are automatically saved.
}
```

For more complex scenarios, like exploring alternative responses or handling tool calls that branch off the main flow, you can use `AttachMessages`. This method lets you specify the parent message ID to which the new messages should be attached, creating a branch in the conversation tree.

```go
func attachMessageBranch(manager *conversation.ManagerImpl) {
	// Assume manager already has some messages, get the ID of the last user message
	conv := manager.GetConversation()
	if len(conv) == 0 {
		fmt.Println("Cannot attach, conversation is empty.")
		return
	}
	// NOTE: This assumes the last message has a valid ID and is the desired parent.
	// In a real application, you might need more robust logic to find the correct parent ID.
	lastMsgID := conv[len(conv)-1].ID 

	// Create an alternative assistant response
	altAssistantMsg := conversation.NewChatMessage(
		conversation.RoleAssistant, 
		"Alternatively, you could consider using mutexes if...",
	)
	
	// Attach the alternative response to the last message
	manager.AttachMessages(lastMsgID, altAssistantMsg)

	// Note: GetConversation() still returns the main (left-most) thread.
	// Accessing branches requires direct tree traversal (not shown here).
}
```


### Retrieving the Conversation for LLMs

Most LLM APIs expect a linear sequence of messages. The `Manager` provides the `GetConversation()` method, which retrieves the current main thread (specifically, the left-most path from the root to the latest message) as a `Conversation` slice (`[]*Message`). This is exactly the format needed for many LLM client libraries.

```go
func getConversationForLLM(manager conversation.Manager) {
	// Get the current linear conversation thread
	messagesForLLM := manager.GetConversation()

	// This slice can now be passed to your LLM API client
	fmt.Printf("Retrieved %d messages for LLM:\n", len(messagesForLLM))
	for _, msg := range messagesForLLM {
		// Truncate long messages for brevity in example output
		contentStr := msg.Content.String()
		if len(contentStr) > 50 {
			contentStr = contentStr[:47] + "..."
		}
		fmt.Printf("- Role: %s, Content: %s\n", msg.Content.GetRole(), contentStr)
	}

	// Example: Prepare for an API call (pseudo-code)
	// apiRequest := &LLMAPIRequest{
	//     Model:    "claude-3-opus",
	//     Messages: messagesForLLM,
	// }
	// response := callLLMAPI(apiRequest)
	// processResponse(response)
}

```

### Saving and Loading (Briefly)

The `Manager` includes a `SaveToFile(filename string)` method to manually save the *current linear conversation* (obtained via `GetConversation()`) to a JSON file. As mentioned, you can also configure autosave during manager creation using `WithAutosave`. Loading conversations currently requires manual implementation by reading the JSON file and reconstructing the `Manager` or `ConversationTree`.

```go
func manualSave(manager conversation.Manager) error {
	filename := fmt.Sprintf("conversation_%s.json", uuid.New().String())
	fmt.Printf("Saving conversation to %s\n", filename)
	err := manager.SaveToFile(filename)
	if err != nil {
		fmt.Printf("Error saving conversation: %v\n", err)
		return err
	}
	fmt.Println("Conversation saved successfully.")
	return nil
}
```

Using the `Manager` provides a structured way to handle the flow of messages in an LLM interaction, maintain the history correctly, and easily retrieve the required message format for API calls.

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
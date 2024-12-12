# Conversation JS Wrapper

The conversation JS wrapper provides a JavaScript interface for working with Geppetto conversations. It allows creating and managing conversations with messages, tool uses, and tool results.

## Usage

```javascript
// Create a new conversation
const conv = new Conversation();

// Add a simple chat message
const msgId = conv.addMessage("user", "Hello, how can I help you?");

// Add a message with options
const msgWithOptions = conv.addMessage("system", "System prompt", {
    metadata: { source: "config" },
    parentID: "parent-message-id",
    time: "2024-01-01T00:00:00Z",
    id: "custom-id"  // optional, will generate UUID if not provided
});

// Add a message with an image
const msgWithImage = conv.addMessageWithImage(
    "user",
    "Here's an image",
    "/path/to/image.jpg"  // supports local files and URLs
);

// Add a tool use
const toolUseId = conv.addToolUse(
    "tool123",
    "searchCode",
    { query: "find main function" }
);

// Add a tool result
const resultId = conv.addToolResult(
    "tool123",
    "Found main function in main.go"
);

// Get all messages
const messages = conv.getMessages();
// messages is an array of message objects, see Message Objects section below

// Get formatted view of a specific message
const messageView = conv.getMessageView(msgId);

// Update message metadata
conv.updateMetadata(msgId, { processed: true });

// Get conversation as a single prompt string
const prompt = conv.getSinglePrompt();

// Convert back to Go conversation object
const goConv = conv.toGoConversation();
```

## API Reference

### Constructor

- `new Conversation()` - Creates a new conversation instance

### Methods

- `addMessage(role: string, text: string, options?: MessageOptions): string` - Adds a chat message with the given role and text. Returns the message ID.
  - Roles: "system", "assistant", "user", "tool"
  - Options:
    ```typescript
    interface MessageOptions {
        metadata?: Record<string, any>;
        parentID?: string;
        time?: string;  // RFC3339 format
        id?: string;    // custom message ID
    }
    ```

- `addMessageWithImage(role: string, text: string, imagePath: string): string` - Adds a chat message with an attached image. Returns the message ID.
  - `imagePath`: Local file path or URL
  - Supports PNG, JPEG, WebP, and GIF formats
  - Maximum file size: 20MB

- `addToolUse(toolId: string, name: string, input: object): string` - Adds a tool use message. Returns the message ID.
  - `toolId`: Unique identifier for the tool
  - `name`: Name of the tool being used
  - `input`: Tool input parameters as a JavaScript object

- `addToolResult(toolId: string, result: string): string` - Adds a tool result message. Returns the message ID.
  - `toolId`: ID matching the tool use
  - `result`: Result string from the tool execution

- `getMessages(): Message[]` - Returns array of message objects (see Message Objects section)

- `getMessageView(messageId: string): string | undefined` - Returns a formatted string representation of the message
  - Returns undefined if message not found
  - Format varies by message type:
    - Chat: "[role]: text"
    - Tool Use: "ToolUseContent{...}"
    - Tool Result: "ToolResultContent{...}"

- `updateMetadata(messageId: string, metadata: object): boolean` - Updates a message's metadata
  - Returns true if message was found and updated
  - Returns false if message not found

- `getSinglePrompt(): string` - Returns the conversation formatted as a single prompt string

- `toGoConversation(): Conversation` - Converts the JS conversation back to a Go conversation object

## Message Objects

The `getMessages()` method returns an array of message objects. Each message has common fields and type-specific fields based on its `type`.

### Common Fields
All message objects include:
```javascript
{
    id: string,          // Unique message ID
    parentID: string,    // Parent message ID
    time: Date,          // Creation timestamp
    lastUpdate: Date,    // Last update timestamp
    metadata: object,    // Additional metadata
    type: string        // Message type: "chat-message", "tool-use", or "tool-result"
}
```

### Chat Message
Type: `"chat-message"`
```javascript
{
    ...common fields,
    role: string,        // "system", "assistant", "user", or "tool"
    text: string,        // Message content
    images?: [{          // Optional array of images
        imageURL: string,
        imageName: string,
        mediaType: string,
        detail: string
    }]
}
```

### Tool Use
Type: `"tool-use"`
```javascript
{
    ...common fields,
    toolID: string,      // Tool identifier
    name: string,        // Tool name
    input: object,       // Tool input parameters
    toolType: string     // Tool type (e.g., "function")
}
```

### Tool Result
Type: `"tool-result"`
```javascript
{
    ...common fields,
    toolID: string,      // Tool identifier
    result: string       // Tool execution result
}
```
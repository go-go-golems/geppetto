# Conversation Manager Integration

Refactored conversation-js.go to use the Manager interface instead of raw Conversation type for better encapsulation and consistency.

- Replaced direct Conversation manipulation with Manager interface methods
- Updated message handling to use Manager's AppendMessages
- Updated message lookups to use Manager's GetMessage
- Maintained backward compatibility with JavaScript API 

# Chat Step Factory Integration with JSConversation

Updated factory.go to use JSConversation for better conversation handling.

- Added support for JSConversation objects as input
- Maintained backward compatibility with legacy message format
- Used JSConversation's AddMessage for proper message handling through Manager 

# Step Object Creation Refactor

Refactored steps-js.go to create step objects directly without global registration.

- Added CreateStepObject function to create step objects directly
- Maintained RegisterStep for backward compatibility
- Updated factory.go to use CreateStepObject instead of RegisterStep
- Removed need for generating unique step names 

# Chat Step Factory Documentation

Added comprehensive documentation for the JavaScript chat step factory bindings.

- Added usage examples and best practices
- Documented Conversation class integration
- Added streaming and Promise-based examples
- Included error handling patterns 

# Chat Step Integration Tests

Added chat step integration tests to demonstrate the JavaScript bindings.

- Added runChatStepTest function to test ChatStepFactory
- Added tests for both Promise and Streaming APIs
- Added example of conversation management with chat steps
- Added flag to control chat step test execution 

# Improved JavaScript Conversation Bindings

Simplified the JavaScript conversation bindings by leveraging Goja's automatic struct mapping capabilities. This provides better type safety and reduces boilerplate code.

- Removed manual method bindings in RegisterConversation
- Changed method signatures to use native Go types instead of Goja-specific types
- Improved error handling by returning proper errors instead of panicking
- Methods now return promises in JavaScript for better async handling 

# Event Loop Termination Improvements

Added proper event loop termination handling to prevent goroutine leaks and ensure clean program shutdown.

- Added done channel for signaling completion in main.go
- Updated runChatStepTest to properly signal completion
- Added documentation for event loop termination patterns
- Improved error handling in JavaScript callbacks 

# Documentation: JavaScript Embeddings API

## Why
Added comprehensive documentation for the JavaScript Embeddings API to help developers understand and effectively use the embeddings functionality in JavaScript applications.

- Added embeddings-js.md with detailed API documentation
- Included usage examples, best practices, and integration patterns
- Added technical details and advanced topics 

# Embeddings Integration Tests

## Why
Added integration tests for the JavaScript embeddings API to verify functionality and provide usage examples.

- Added runEmbeddingsTest function to test embeddings functionality
- Added mock embeddings provider for testing
- Added semantic similarity search example
- Added flag to control embeddings test execution 

# Settings-Based Embeddings Factory

Added a new factory for creating embeddings providers that uses the existing chat settings infrastructure.

- Added SettingsFactory to create embedding providers from StepSettings
- Supports both OpenAI and Ollama providers
- Reuses existing API keys and base URLs from chat settings
- Provides consistent configuration across chat and embeddings

# Dedicated Embeddings Settings

Added dedicated embeddings settings to separate embeddings configuration from chat settings.

- Added new EmbeddingsSettings type with model, provider type, and dimensions
- Updated SettingsFactory to use dedicated embeddings settings
- Added embeddings parameter layer with defaults
- Integrated embeddings settings into StepSettings

# Embeddings Settings Reorganization

Reorganized embeddings settings to match the chat settings structure.

- Moved embeddings settings to dedicated settings-embeddings.go file
- Added embeddings.yaml flags file with proper parameter definitions
- Updated StepSettings to use the new EmbeddingsSettings type
- Added embeddings layer to UpdateFromParsedLayers

# Embeddings Test Integration

Updated embeddings test to use the new settings-based provider.

- Replaced mock provider with real provider from settings
- Added proper error handling for provider creation
- Maintained existing semantic similarity test functionality
- Added support for both OpenAI and Ollama providers

# Enhanced JavaScript Embeddings API

Added async/callback-based methods to the JavaScript embeddings API for better integration with event-loop based applications.

- Added generateEmbeddingAsync for Promise-based embedding generation
- Added generateEmbeddingWithCallbacks for callback-based embedding generation
- Updated embeddings documentation with new API methods
- Improved error handling and type safety

# Async Embeddings Test Wrapper

Improved embeddings test structure by wrapping it in an async function for better async/await support and error handling.

- Wrapped embeddings test in async function
- Added proper error signaling through done channel
- Updated main loop to wait for embeddings test completion
- Maintained existing test functionality

# Add Conversation HashBytes method

Added HashBytes() method to Conversation type to generate deterministic hashes of conversation content. This enables caching, comparison and verification of conversation content.

- Added HashBytes() method that generates xxHash of all message content
- Handles all message types including chat messages, tool use, tool results
- Includes image content and metadata in hash calculation

# Add fast hash method for Conversation caching

Changed HashBytes() method to use xxHash instead of SHA-256 for better performance when used as a cache key.

- Switched from crypto/sha256 to xxhash for faster hashing
- Maintained same content coverage (messages, tools, images, metadata)
- Optimized for caching use case

# Add conversation caching to LLMHelper

Added caching support to LLMHelper to avoid redundant LLM calls for identical conversations.

- Added ExecuteCached method that uses conversation hash as cache key
- Added thread-safe in-memory cache with RWMutex
- Added ClearCache method to reset the cache
- Cache stores both results and errors

# Add caching support for embedding providers

Added a caching wrapper for embedding providers to avoid redundant embedding generation.

- Added CachedProvider that wraps any Provider implementation
- Added thread-safe in-memory cache with RWMutex
- Added ClearCache method to reset the cache
- Updated SettingsFactory to support creating cached providers

# Update embedding cache to use LRU

Changed the embedding cache to use a Least Recently Used (LRU) eviction policy to prevent unbounded memory growth.

- Added size limit to embedding cache with LRU eviction
- Added Size() and MaxSize() methods for monitoring
- Used container/list for efficient LRU implementation
- Updated factory to support configuring cache size

# Update LLMHelper to use LRU cache

Changed the LLMHelper cache to use a Least Recently Used (LRU) eviction policy to prevent unbounded memory growth.

- Added size limit to conversation cache with LRU eviction
- Added Size() and MaxSize() methods for monitoring
- Used container/list for efficient LRU implementation
- Updated NewLLMHelper to support configuring cache size

# Add LLM Toggle for Query Testing

Added ability to toggle LLM processing in query testing. This allows testing raw queries without LLM processing.

- Added UseLLM flag to PromptData to control LLM processing
- Added UI toggle in prompt details form
- Updated runner to support bypassing LLM processing
- Added LLM toggle state persistence in prompt metadata
- Simplified prompt update handling by using PromptData struct

# SSE Container Visibility and EchoStep Initialization

Fixed initialization of EchoStep to properly handle event publishing and improved UI by only showing the SSE container after starting a chat.

- Fixed EchoStep initialization by using NewEchoStep() to properly initialize the subscriptionManager
- Updated UI to hide SSE container initially and show it only after starting a chat
- Added proper event handling for chat start

# Improved Client ID Handling and Logging

Enhanced the web UI server with better client ID handling and comprehensive logging.

- Changed /start endpoint to create and return client ID
- Added zerolog for structured logging throughout the server
- Added detailed logging for client connections, steps, and events
- Improved error handling and reporting
- Updated UI to handle new client ID flow

# Enhanced Step and Message Logging

Added comprehensive logging for step execution and message handling to improve debugging and monitoring.

- Added detailed logging for step startup and conversation content
- Added message-level logging for router events with message IDs and types
- Added result processing logging with counts and content
- Added success/failure logging for message delivery to clients

# Improved HTMX Integration and Session Persistence

Enhanced the web UI to use more traditional HTMX patterns and support session persistence.

- Changed /start endpoint to return HTML template instead of JSON
- Added client ID to URL for session persistence
- Added support for reconnecting to existing sessions
- Removed JavaScript-based client handling
- Improved template structure with partial templates

# URL Handling in Web UI

Improved URL handling in the web UI to update the URL when starting a new chat using HTMX's built-in URL handling capabilities.

- Added HX-Push-Url header in /start handler to update URL with client ID

# Add SSE Backend Tutorial

Added comprehensive tutorial explaining how to build a Server-Sent Events backend for streaming steps.

- Added detailed explanation of system architecture
- Added code examples and best practices
- Added implementation steps and testing guide
- Added common issues and solutions
- Added next steps for future improvements

# Expanded SSE Backend Tutorial

Added detailed technical sections to the SSE backend tutorial.

- Added in-depth explanation of event router setup and configuration
- Added step-by-step guide to step creation and execution
- Added detailed event flow and handler implementation
- Added event template examples and usage
- Added comprehensive error handling guide

# Code Organization: Split Web UI Components

Improved code organization by splitting the web UI components into separate files for better maintainability:

- Split `main.go` into multiple files:
  - `client.go`: Contains SSEClient related code
  - `server.go`: Contains Server struct and core methods
  - `handlers.go`: Contains HTTP handlers
  - `events.go`: Contains event handling code
  - Simplified `main.go` to only handle initialization

# Add Chat Input to Web UI

Added a proper chat interface to allow users to send messages through a text input field.

- Added chat input form to index.html template
- Added /chat endpoint to handle chat messages
- Kept existing SSE event handling for responses

# Improved Chat UI with Message Handling

Refactored the web UI to handle chat messages properly and improve code organization.

- Moved step management from server to client
- Removed /start endpoint in favor of unified /chat endpoint
- Added proper message handling in chat steps
- Improved client-side state management

# Refactor Step Management in Web UI

Moved step management responsibility from server to client for better encapsulation.

- Moved step creation and event handling to SSEClient
- Added per-client logging with zerolog
- Removed step management from Server struct
- Improved client initialization with template dependency

# Improve Event Handler Registration in Web UI

Moved event handler registration to client initialization for better lifecycle management.

- Moved event handler setup from CreateStep to NewSSEClient
- Added router dependency to SSEClient constructor
- Simplified CreateStep to only handle step creation
- Improved client initialization flow

# Simplify Chat UI

Simplified the chat UI by removing the explicit start button.

- Removed "Start New Chat" button
- Always show chat input area
- First message automatically starts a new chat
- Improved initial UI state

# Conversation Manager Integration in Web UI Client

Added conversation management capabilities to the SSE client in the web UI to maintain chat history and provide better conversation state management.

- Added conversation.Manager field to SSEClient struct
- Initialized conversation manager in NewSSEClient
- Updated StartStep to track messages in conversation manager
- Added GetConversation and AddMessage helper methods
- Integrated assistant responses into conversation history

# Simplified Message Handling in Web UI

Simplified message handling in the web UI by moving all conversation management to the client and removing redundant message handling in handlers.

- Renamed StartStep to SendUserMessage for better clarity
- Removed message handling from handlers.go
- Updated client to use conversation manager for message history
- Improved error handling and logging
- Simplified chat endpoint implementation

# Improved Chat UI with User Messages

Enhanced the chat UI to display user messages and provide better interaction feedback.

- Added user message template with distinct styling
- Added automatic input clearing after message submission
- Improved message flow with proper conversation view
- Added timestamps to all messages
- Maintained existing event handling for assistant responses

# Improved Message Streaming in Chat UI

Enhanced the chat UI to better handle message streaming and maintain conversation flow.

- Separated static user messages from streaming assistant responses
- Added message groups to keep related messages together
- Improved SSE event targeting for cleaner streaming
- Maintained proper conversation history display

# Enhanced Conversation History in Chat UI

Improved the chat UI to show the full conversation history and maintain context.

- Added conversation history rendering in message containers
- Updated templates to handle both user and assistant messages
- Improved message grouping for better readability
- Fixed conversation type handling in templates

# Improved Template Handling in Chat UI

Enhanced the chat UI templates with better data handling and formatting.

- Added sprig template functions for improved template functionality
- Fixed timestamp handling in conversation history
- Updated message data passing to use proper dictionary functions
- Improved template consistency across all message types

# Refactor Web UI Handlers into Server Methods

Improved code organization by moving handlers from standalone functions into Server struct methods. This change:
- Makes the code more object-oriented and encapsulated
- Reduces global state by keeping all server-related functionality in the Server struct
- Simplifies handler registration with a single Register method
- Removes the need for a separate handlers.go file

## OpenAI Chat Completion Metadata Extraction

Added functionality to extract and include metadata from OpenAI chat completion stream responses in events. This includes model information, system fingerprint, prompt annotations, usage statistics, and content filter results.

- Added independent metadata types to avoid direct OpenAI package dependencies
- Added mapstructure tags for flexible metadata conversion
- Added helper function to convert OpenAI content filter results
- Updated chat step to handle metadata extraction errors
- Added support for JailBreak and Profanity filter results
- Added detailed token usage information including audio, cached, and reasoning tokens

# Improved Usage Tracking in Claude Content Block Merger

Added proper usage tracking in the Claude ContentBlockMerger to maintain accurate token counts throughout streaming responses.

- Added dedicated usage field to ContentBlockMerger struct
- Updated usage tracking to maintain running totals from streaming events
- Ensured consistent usage metadata format across all events

# Unified Event Metadata Across Chat Steps

Moved common metadata fields from step metadata to event metadata for better consistency and accessibility.

- Added common fields (engine, temperature, topP, maxTokens, stopReason, usage) to EventMetadata
- Updated ContentBlockMerger to track metadata in both event and step metadata
- Updated OpenAI chat step to use new event metadata fields
- Improved metadata extraction from OpenAI responses
- Updated printer to use event metadata instead of step metadata for common fields
- Ensured consistent metadata format across all chat events
- Improved logging of metadata fields

# Optional Usage in EventMetadata

Made Usage field in EventMetadata optional by using a pointer to improve flexibility and better handle cases where usage information is not available.

- Updated EventMetadata to use *Usage instead of Usage
- Modified content-block-merger.go to handle nil Usage
- Updated chat-step.go to use pointer for Usage assignments

# Improved Input Token Tracking in Claude Content Block Merger

Enhanced the Claude ContentBlockMerger to properly maintain input token counts across streaming events, since input tokens are only received in the initial start event.

- Added inputTokens field to ContentBlockMerger to persist input token count
- Updated updateUsage to maintain input tokens from start event
- Ensured consistent input token reporting in usage metadata across all events
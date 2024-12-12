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
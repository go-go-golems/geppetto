# Refactor Command Execution with RunContext

Introduced a new RunContext pattern to simplify and clarify command execution:

- Created RunContext to encapsulate all run-related state and settings
- Introduced RunOptions for flexible configuration
- Separated core functionality from UI/terminal features
- Added clear modes for blocking, interactive, and chat execution
- Simplified the interface for programmatic usage 

# Changelog

## Caching Layer for Chat Steps

Added a caching layer for chat steps to improve performance by storing and reusing chat responses.

- Added `CachingStep` that wraps another step and caches its responses
- Added disk-based caching with LRU eviction based on file modification time
- Support for multiple messages in a single response
- Configurable cache size and entry limits
- Added `MockStep` for testing that returns messages in a round-robin fashion 
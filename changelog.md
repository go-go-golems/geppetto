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

## Fix nil pointer panic in TranscribeCommand

Added a nil check in TranscribeCommand.RunIntoGlazeProcessor to prevent panic when command is nil.

- Added nil check at the beginning of RunIntoGlazeProcessor 

## Enhanced Audio Transcription Support

Added streaming and timestamp support to the OpenAI transcription client:

- Added streaming mode for real-time transcription processing
- Added support for word and segment-level timestamps
- Added progress tracking for long transcriptions
- Improved error handling and logging
- Added context support for cancellation 

## Improved Transcription API

Refactored the transcription API for better usability:

- Split into separate blocking and streaming functions
- Added functional options pattern for both client and transcription options
- Added sensible defaults for all options
- Improved type safety with specific return types
- Simplified client initialization with optional configuration 

## Improved Transcription Options API

Enhanced the transcription options API for better usability:

- Added individual option constructors for each timestamp granularity (word/segment)
- Added individual option constructors for each output format (JSON/Text/SRT/VTT)
- Improved type safety with specific option constructors
- Added sensible defaults for formats and chunk sizes 

## Improved Transcription Output Format

Enhanced the transcription output handling:

- Removed redundant with-segments flag
- Added automatic output format selection based on timestamp granularity
- Added word-level output when word timestamps are requested
- Standardized output field names (text instead of response)
- Added fallback to full text when no segments or words are available

## Added Advanced Transcription Callbacks and Error Handling

Enhanced the transcription API with advanced control and monitoring:

- Added detailed progress tracking with time estimates and speaker info
- Added callbacks for speaker changes, silence detection, and noise levels
- Added comprehensive error handling with retries and fail-fast options
- Added rate limiting controls for API usage optimization
- Added structured error handling and reporting 

## Enhanced Transcribe Command

Updated the transcribe command with comprehensive options:

- Added all transcription options as command flags with proper types
- Added format selection with choice parameter
- Added timestamp granularity selection with choice list
- Added speaker diarization options
- Added performance and error handling options
- Added rate limiting controls
- Improved error handling and logging
- Added proper parameter validation 

## Documentation for Embeddings Tag Function

Added comprehensive documentation for the !Embeddings tag function, including:
- Basic usage examples
- Configuration options for OpenAI and Ollama providers
- Default settings and environment variables
- Error handling guidelines 

## Embeddings Caching Support

Added configurable caching support for embeddings with both memory and disk-based options.

- Added caching configuration to embeddings settings with none/memory/disk options
- Implemented memory and disk caching options for embeddings
- Cache is disabled by default (using "none" type)
- Configurable cache size and entry limits 

## Unified Caching Configuration

Standardized caching configuration across embeddings and chat:

- Added consistent caching options (none/memory/disk) for both embeddings and chat
- Simplified configuration by using cache type to control enabling/disabling
- Added shared configuration parameters for cache size and entries
- Cache is disabled by default (using "none" type) 
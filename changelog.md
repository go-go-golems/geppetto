# Refactor Command Execution with RunContext

Introduced a new RunContext pattern to simplify and clarify command execution:

- Created RunContext to encapsulate all run-related state and settings
- Introduced RunOptions for flexible configuration
- Separated core functionality from UI/terminal features
- Added clear modes for blocking, interactive, and chat execution
- Simplified the interface for programmatic usage 

# History Browser TUI Application

Added a terminal-based UI application for browsing history recordings:

- Created a new charmbracelet bubbletea-based TUI in pinocchio/cmd/history-browser
- Implemented browsing functionality for history recording files with proper pagination
- Added split-pane UI with list and detail views for better navigation
- Created a proper HistoryFile interface for typed file access
- Implemented concrete HistoryFileImpl with details extraction
- Added real-time file information display in side panel
- Added formatted styling for better readability
- Implemented viewport scrolling for file details
- Added formatted display of filenames with timestamps
- Implemented Cobra command-line interface with directory selection
- Set default history path to ~/.pinocchio/history
- Created a simple and intuitive navigation interface
- Included README with usage instructions

## Improved History Browser UI

Enhanced the history browser user interface for better usability:

- Fixed file listing truncation issues in the left pane
- Added fixed-width styling to prevent title wrapping
- Improved list item display with consistent formatting
- Added automatic file information display when navigating
- Enhanced modal view with comprehensive file details
- Updated the documentation with new behavior and features

## Fixed History Browser Bugs and Improved Layout

Fixed critical bugs and improved the layout of the history browser:

- Fixed panic caused by comparing uncomparable HistoryFileImpl types
- Increased list pane width for better file name visibility
- Improved selection change detection using path comparison
- Enhanced overall UI stability and reliability

## Added HistoryFile Comparison Support

Enhanced the HistoryFile interface with proper comparison functionality:

- Added Compare method to HistoryFile interface
- Implemented Compare method in HistoryFileImpl
- Updated selection change detection to use Compare method
- Improved type safety in file comparisons

## Fixed History Browser Selection Comparison

Fixed critical bug in history browser selection handling:

- Removed direct interface comparison that was causing panic
- Improved selection change detection logic
- Added proper nil handling for selections
- Enhanced type safety in selection comparison

## Enhanced History Browser UI

Improved the history browser interface for better usability and aesthetics:

- Added distinct background color for selected items in the list
- Implemented proper overlay modal using bubbletea-overlay package
- Enhanced visual hierarchy with better color contrast
- Fixed list item styling to prevent wrapping issues
- Improved overall UI consistency and readability

## Added History Browser Tutorial Documentation

Created comprehensive tutorial documentation for building TUI applications:

- Added detailed explanation of the bubbletea application structure
- Documented the generic data interface pattern for reusable list items
- Provided examples for different data types (files, users, products)
- Explained split-pane layout implementation techniques
- Provided guidance on implementing modal overlays
- Detailed the control flow and selection change detection
- Included styling and UI design patterns
- Added command-line integration examples
- Created a reusable reference for building similar TUI applications

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

## Preserve pointer types in reflect package

Changed the behavior of StripInterface and related functions to preserve pointer types while stripping interfaces. This allows for more accurate type handling when dealing with pointer types that contain interfaces.

- Updated `StripInterface` to preserve pointer types while stripping interfaces
- Updated `StripInterfaceFromValue` to maintain pointer structure
- Added comprehensive test cases for pointer type handling
- Fixed behavior for nested pointers and interfaces 

## Empty Profile File Check

Added check for empty or whitespace-only profile files in GatherFlagsFromProfiles to prevent YAML parsing errors and provide better feedback.

- Added check for empty profile files with warning log
- Improved error handling for profile file reading 

## Skip Socket Files in File Watcher

Added functionality to skip socket files in the file watcher to prevent potential issues:

- Added socket file detection in addRecursive method
- Skip socket files during initial path processing
- Skip socket files during directory walk
- Added debug logging for skipped socket files

## Fixed Credential Editor Styling

Fixed styling issues in the credential editor application:

- Implemented custom delegate for proper list item styling
- Fixed selected item highlighting with correct styling approach
- Updated documentation to reflect the proper styling technique
- Added comprehensive package import information
- Improved tutorial with a dedicated section on custom list styling

## Credential Editor Application

Created a terminal-based user interface (TUI) for viewing and editing credentials.

- Created main.go with a Bubbletea-based TUI application
- Implemented a split-pane layout with a list of credentials on the left and details on the right
- Added modal overlay for viewing credential values
- Integrated Huh library for form-based credential editing
- Created README.md with usage instructions
- Created TUTORIAL.md with detailed implementation explanation

## Fix scrolling layout issues in edit-credentials TUI

Fixed a bug in the edit-credentials TUI application where scrolling the list would break the layout. The changes include:

- Added proper window size handling to update list and viewport dimensions when the window is resized
- Improved selection change detection using a more robust approach similar to history-browser
- Added fixed-width styles for list items to prevent wrapping issues
- Reorganized the list update logic to better handle selection changes

These changes ensure the layout remains stable when scrolling through the credential list.

## Added Debug Logging to Edit Credentials Application

Enhanced the edit-credentials TUI application with comprehensive logging for debugging:

- Added zerolog logger initialization with output to /tmp/edit-credentials.log
- Added detailed logging for window resize events with dimension tracking
- Added logging for selection changes and list updates
- Added logging for credential updates and modal operations
- Added logging for view rendering with dimension information
- Added application lifecycle logging (startup, shutdown)

This logging helps diagnose layout issues when scrolling through the credential list and provides insights into the application's internal state changes.

## Enhanced Debug Logging in Edit Credentials Application

Improved the debug logging in the edit-credentials TUI application for better troubleshooting:

- Added caller information to log entries for easier stack trace analysis
- Added list index tracking to monitor selection state changes
- Added item count logging to track list size
- Enhanced key press logging with selection context
- Added detailed index change tracking for list navigation
- Added comparison of old and new indices during selection changes
- Improved rendering logs with selection state information

These enhancements provide more context for debugging layout and scrolling issues by tracking the relationship between list indices, selections, and UI rendering.

## Fixed Modal Rendering in Edit Credentials Application

Fixed a critical bug in the edit-credentials TUI application that was causing garbled display when scrolling:

- Restored the mode-based rendering logic in the View() function
- Fixed proper handling of different UI modes (normal, modal, form)
- Ensured correct overlay rendering for modals and forms
- Fixed the display when switching between different credentials

This fix ensures that the UI correctly renders the appropriate view based on the current mode, preventing display issues when navigating through the credential list or when opening modals.

## Fixed Dimension Calculations in Edit Credentials Application

Fixed critical dimension calculation issues in the edit-credentials TUI application that were causing layout problems:

- Adjusted list and viewport height calculations to account for borders and padding
- Fixed info pane width calculation to use the actual rendered list width
- Reduced component heights to ensure they fit within the window
- Added additional logging of window dimensions vs. rendered dimensions
- Improved dimension calculations to prevent content overflow
- Fixed the relationship between list width and info pane width

These changes ensure that the UI components fit properly within the terminal window and maintain their layout when scrolling through the credential list.

## Further Improved Layout in Edit Credentials Application

Made additional improvements to the edit-credentials TUI application to ensure proper layout and prevent content overflow:

- Made credential format more compact to reduce content height
- Replaced fixed height with MaxHeight to prevent content overflow
- Increased spacing between components to prevent overlap
- Added warning logging when content exceeds window dimensions
- Further reduced component heights to ensure they fit within the window
- Increased horizontal spacing between list and info pane

These changes ensure that the UI components maintain proper proportions and fit within the terminal window, even with varying content sizes.

## Removed Redundant Border Styling in Edit Credentials Application

Simplified the UI styling in the edit-credentials TUI application by removing redundant border styling:

- Removed duplicate border styling from the viewport since it already provides its own border
- Removed unnecessary infoPane wrapper styling when rendering the viewport
- Added padding to the viewport style to maintain proper spacing
- Simplified the rendering of the info content by using the viewport view directly
- Improved the overall appearance by eliminating double borders

These changes create a cleaner UI with proper borders while maintaining the same functionality.

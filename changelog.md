# Refactor Command Execution with RunContext

Introduced a new RunContext pattern to simplify and clarify command execution:

- Created RunContext to encapsulate all run-related state and settings
- Introduced RunOptions for flexible configuration
- Separated core functionality from UI/terminal features
- Added clear modes for blocking, interactive, and chat execution
- Simplified the interface for programmatic usage 
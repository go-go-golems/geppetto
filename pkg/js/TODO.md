- Move RegisterEmbeddings as a method on RuntimeEngine
- Pass in the runtime engine context to the javascript wrappers so that cancellation
- remove WithCallbacks for Embeddings
- Remove JSConversation.runtime

- Does the result of the chatStep need to be a pure javascript object? Can't we expose the go conversation.Message object?

- use errgroup for all goroutines

- the RegisteRPublisher for each step execution if we reuse steps will grow over time because we keep registering step.{StepID} without removing it
    - ultimately this is a rewrite of the step abstraction.

- Make CreateWatermillStepObjerct a method on the RuntimeEngine

## Main thing to do 

- We want to run a step in the background but also have an onEvent callback
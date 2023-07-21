# Context Management

## Brainstorm 2023-07-21

I've been thinking about a way to do context management for LLM prompting for a while,
 and prototyping things in org-ai in org-mode in emacs as well. 

Here's a list of things to incorporate in context management:
- previous messages
- snippets with name
- outside files
- live sql queries
- live glazed.Commands more so
- saved prompts and messages
- dynamically recomputed programming language generated snippets (thing randomized words)
- oak queries (see glazed queries)
- extract / postprocessed results of messages

I want the context manager to be a general purpose tool, not just something for the use
of geppetto programs and pinocchio. My idea is to make it a server that can be easily 
queried from other software, and offer a set of rich utility verbs.

For now, I just need to deal with storing and retrieving a message history,
in fact because the chat mode for pinocchio is not implemented yet, this will just
be a set of messages and a system prompt loaded on startup from the CLI or from the
command itself. The flag to set the system prompt and the flag to load a chat history
are flags that need to be in the geppetto layer (which I think doesn't really exist atm)?

Additional ideas:
- tags
- search and post processing of search results

## First design (2023-07-21)

```go
package foo

import "time"

type Message struct {
	Text string
    Time time.Time
    Author string
}

```

- GetMessages() 
- SetMessages(messages []Message)
- AddMessages(messages []Message)
- SetSystemPrompt(prompt string)
- GetSystemPrompt()

Static helper methods:
- LoadFromFile() []*Message
- SaveToFile([]*Message)

In the future, this could potentially all be async, but for now, blocking is enough.
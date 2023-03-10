# Goals of the project

## Changelog 

- 2023-01-22 - Initial version

## Overview

This document contains the overall goal of the project, from which everything else is 
derived. It also serves as the working document to delineate the smaller parts to then be
implemented.

## Goal

The goal of geppetto is to provide the best possible experience for a developer to build 
high-quality production software leveraging LLM technology (OpenAI in particular). 

This means that it should:
- provide comprehensive APIs
  - not just simple chain APIs, but abstractions that support complex application ideas 

## Comprehensive APIs 

Everything is a monad, and so is an LLM application. In contrast to a pure chain model,
potentially with an agent executor, we consider the different steps that are necessary for 
an app from the start. While I use the evil M word here, we are using golang, so it's not
like we are going to switch to a functional style overloaden with maths concepts. Instead,
the abstraction is helping us guide the APIs we provide while ensuring that they will compose
cleanly.

Here are the "tricky" things to take into account:

- individual steps of the application can be doing IO / be asynchronous
  - an agent might do a websearch, or a database query. If we are serving clients over HTTP, we need 
to be able to run this IO in the background.
  - if we are running UIs, we need to provide introspection capabilities to provide a smooth UI experience 
- we want to be able to suspend our state
  - for example, if we build a webapp, or maybe a chatbot, we need to be able to let
agents resume after hours of inactivity
- we want to be able to cleanly interrupt any ongoing process 
  - if only purely while debugging, to allow us to pause in case a certain threshold of tokens
has been reached
- we want to provide error handling that allows for restarts
  - if we are deep into an agent interaction and have built up significant state, we want to
be able to restart in case of an error. This especially due to the non-determinism and use of
natural language
- individual step behaviours can be complicated 
  - For example, we might want to loop until a stop condition is reached
  - We might want to switch to a completely different agent model once a specific step has been reached
  - we might want to unwind completely in case we reached a dead end, and restart from a previous step

Of course, the real power of a monadic approach is that these complexities can be composed easily and 
without much plumbing. This will allow us to reuse and compose complicated pieces of machinery
either in even more complex settings, but also easily extract individual components for unit testing
or training / examination. 

Furthermore, the easy composability means that we can share complex LLM based application without
having to spend time refactoring existing monitoring / ui / debugging / testing setups.

## Design brainstorm 2023-01-23

![image](https://user-images.githubusercontent.com/128441/214211444-1646c47f-5d14-45ba-9e7c-c9c06a21c584.png)


## Sketch 1: Task based

The first try at the API is going to be based around the concept of asynchronous steps that can be composed,
and can be introspected. Cancellation, serialization and observability should be able to be bolted on
without too much trouble once the basics are set. How human resolution (RESTART in lisp, ResolutionRequest in Sauron)
should be handled is open.

I am not sure what to do about structured introspection, but it will probably be a tree of semi structured data.

```go
package pkg

import "context"

type Nothing struct{}

type Result[T any] struct {
  value T
  err   error
}

// Step represents one step in a geppetto pipeline
type Step[A, B any] interface {
    Start(ctx context.Context, a A) error
	GetOutput() <-chan Result[B]
	GetState() interface{}
	IsFinished() bool
}
```

### Common steps

- `SimpleStep` : just wrap a function
- `PipeStep` : pipe two steps together
- `TemplateRenderStep` : render a template
- `TemplateFileRenderStep` : render a template from a file
- `HTTPStep` : perform an HTTP request
- `GPT3Step` : perform a GPT3 request
- `CodexStep` : perform a Codex request

## Brainstorming

### Extensible scripting language

These are all widely starred and active:

- https://github.com/antonmedv/expr - single line, type checked, non-turing complete expr. Has a javascript code editor
- https://github.com/google/cel-go - another non turing complete language
- https://github.com/dop251/goja - javascript implementation in golang
- https://github.com/yuin/gopher-lua - lua implementation
- https://github.com/mattn/anko - less active, less appealing, but mattn
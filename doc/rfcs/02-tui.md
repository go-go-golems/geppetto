# Interactive terminal UI for pinocchio

## Changelog

- 2023-01-22 - Initial brainstorm

## Overview

Because LLMs at their core are so interactive, and based on human language, I feel that instead
of making pinocchio a CLI tool only, it should actually be centered around interaction.

## Widgets

The technology will be used a lot for chatbots, so I figured that a little chat textinput at the
bottom could be fun, with / as a command key being linked to the different commands.

For tab completion and other selections, little menus could be shown appearing on top, and 
I could maybe gain some inspiration from the autocompleter in emacs or vim, for example, to make
them both nice UX wise and also look slick.

For longer prompts, it should be possible to use a full textarea, and of course vim bindings would 
be fun, but that really is a completely different undertaking. 

Of course, a console that shows the output of the LLM communication and agents would be a plain scrollable 
view, that maybe leverages markdown rendering? Or it could maybe be visually more interesting to render
it all in bubbletea, but then export the session to markdown (or in fact, to a whole bundle of files
that contain the metadata and the structured data of the interaction and whatever else might be nice to have).

In another window, we could show some status variables (for example, token usage, status of potential requests 
to the backend, status of the currently running agents). This supposes that there is some kind of central orchestraor
that knows about all the agents currently registered.

## Commands

Similar to sqleton, I think it would be really neat if full apps could be described in yaml, along with
their input (and output?) parameters.

Because commands describe their inputs, we can use that to automatically populate the /command parser,
but also generate graphical forms (and views) that can be used by a user to graphically edit their settings.

The affordances of sqleton (saving alises to disk) should be built upon. Because the system here is monadic, chaining 
commands should be easy from the UI too.


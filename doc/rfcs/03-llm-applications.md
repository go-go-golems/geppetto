# Potential LLM applications 

## Changelog 

- 2023-01-22 - Initial version

## Overview 

This document contains a list of LLM applications that would be fun to build 
with geppetto and potentially integrate into pinocchio.

Besides just having a repository of ideas to build, the list of applications should guide 
the design of the library, which is why pretty complicated apps are described here. If we 
build a framework that doesn't allow for the future integration of these more complex ideas,
we might code ourselves into a tricky corner.

## Ideas

### SQL query explainer

The idea here is adding explanation and documentation metadata for example to the output
of sqleton or glazed queries. We could feed the LLM some metadata about the schema (or use
an agent scheme that allows the LLM to prompt itself for information it needs), as well
as the query (or just the query template) and the input values that were fed to the sqleton command.

### Search heuristic provider 

The idea is to use the LLM to provide "intelligent" heuristics when doing things like beam search.
If the search domain is structured and related to a human domain, then the LLM might be able to
provide pretty well reasoned (potentially with explanation) choices for which branches to prune
and which to consider.

### Doing log detective work

Provided with an event log and potentially some commands to ask for additional domain specific 
information, ask a LLM to figure out why something might have happened.

### IDE refactor daemon

Provided an IDE control language, that could be implemented in a plugin (say, extract XYZ to variable, 
rename method to Y, move file to dir, etc...). We would provide again a list of agent methods that allow
the agent to inspect the repository.

### Cyber-physical systems

Maybe there is something interesting to be done by integrating real world sensors into the application.
Of course, hooking up real world actors to an LLM might not be the best idea.
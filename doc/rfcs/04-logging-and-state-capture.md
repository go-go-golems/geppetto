# Logging and point capture

An important part of building and validating LLM applications is being able to 
see the intermediate steps that happened, along with all the metadata that was used.

Logging and point capture goes hand in hand with always having the recording on.

There should be tools available to quickly:

- list the snapshot points in an application run
- restart an application from a snapshot
- extract metrics into tabular data
- live dashboard of a running application
- metrics endpoint if the application is a service for observability

### Brainstorm ideas

- create an alias for a snapshot point quickly (?)

## Goals of logging

- Retrace what happened
- Restart an application from an arbitrary point
- Compute costs and other metrics
- Debug LLM applications

## List of data to capture for multiple steps

### Template render step

- date
- input data, how the input data was passed in (?)
- input prompt template
- output prompt 

### OpenAI Step logging

- date, API version
- input prompt
- client settings
- completion step settings
- outputs
- metrics (from the API?)

### Parsing step

- input text
- parser settings / grammar / checkpoint



## Logging mechanisms

I am not sure if a channel is the best way to emit logging data, because they are blocking / need to be service
(or buffered), so maybe something like watermill router is better.

Let's look at some observability solutions later.

For now, use zerolog logger.
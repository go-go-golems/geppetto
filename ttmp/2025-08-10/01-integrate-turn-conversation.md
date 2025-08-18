I want to replace the conversation as the base data structure to interact with the LLM 
so that middlewares have access to richer data.

I vibecoded a run/turn/conversation/message sql ent model yesterday as a prototype. I now want
to integrate the concept into geppetto. 

A run is basically a whole sequence of interactions with a chatbot / llm. 
A run has multiple turns (a turn is basically one interaction with the llm).
A turn has data (which is the actual data of the application), a set of tools, a conversation and metadata.
A middleware can transform a turn into a modified turn (adding/computing data, adding/removing tools, etc...).
RunInference as it is should now take a turn and return a turn (which would wrap the current conversation).

We do want to keep the entgo structure to store and retrieve a full run from disk.

I want to have a middleware that stores the current mode of the agent.
Modes are linked to: 
 - a set of tools
 - a prompt to be inserted
The modes definitions are stored in sql.


The agent tool middleware besides looking up the tools + prompt, also inserts 
a prompt that says how to switch modes (through a <mode-switch> block).
It also parses the <mode-switch> block and switches to the corresponding mode.

Do we want to add a middleware that adds the turn id + run id to the inference. 
We can use the run_id + turn_id to store the current mode / mode transitions.
This allows us to visualize. 

We can add a explanation block to the mode switch (chain of thought prompting).
I want a middleware that you give a sqlite.db and it adds a tool to execute queries to the turn.

It will expose the schema as a prompt message, by reading the schema and transforming it to sql. (without the _prompts table).

It will also look for a _prompts table, and add these prompts to the turn. It can also be overridden when loading the table (or provide a default that will be inserted in _prompts if missing).
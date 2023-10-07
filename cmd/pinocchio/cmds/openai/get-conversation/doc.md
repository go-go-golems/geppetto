The `get-conversation` command is designed to seamlessly convert GPT HTML into markdown format, catering to both individual users and developers who require a systematic transformation of their ChatGPT transcripts.

**Key Features:**

- **Markdown Conversion**: Efficiently transforms GPT HTML content into clean and organized markdown.

- **Customizable Outputs**: Offers options for concise representation (`--concise`), inclusion of only the assistant's responses (`--only-assistant`), and more to tailor the output based on user preferences.

- **Metadata Integration**: The `--with-metadata` flag ensures the inclusion of conversation metadata, preserving detailed context.

- **Role Renaming**: The `--rename-roles` option allows users to modify default role names for a more customized transcript representation. This is useful when asking a LLM to process the transcript, as it easily gets confused when participants in a conversation are called "user" and "assistant".

- **JSON Output Options**: Provides flexibility in JSON output presentation. Users can opt for a comprehensive view with `--full-json` or focus solely on the conversation using `--only-conversations`.

- **Developer-Centric Features**: Enables extraction of source blocks with `--only-source-blocks`, merges them using `--merge-source-blocks`, and supports inlining conversations as comments in source blocks via `--inline-conversations`.

The `get-conversation` command offers a robust solution for those seeking a structured and customizable way to convert
and manage their ChatGPT transcripts, making it a valuable tool in the open-source toolkit.
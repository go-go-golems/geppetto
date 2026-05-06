##### ModelsExpand Collapse

CompactedResponse object { id, created\_at, object, 2 more }

id: string

The unique identifier for the compacted response.

created\_at: number

Unix timestamp (in seconds) when the compacted conversation was created.

formatunixtime

object: "response.compaction"

The object type. Always `response.compaction`.

output: array of [Message](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20message%20%3E%20\(schema\)) { id, content, role, 3 more } or object { arguments, call\_id, name, 4 more } or object { id, arguments, call\_id, 4 more } or 22 more

The compacted list of output items.

One of the following:

Message object { id, content, role, 3 more }

A message to or from the model.

id: string

The unique ID of the message.

content: array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [TextContent](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20text_content%20%3E%20\(schema\)) { text, type } or 6 more

The content of the message

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

TextContent object { text, type }

A text content.

SummaryTextContent object { text, type }

A summary text from the model.

text: string

A summary of the reasoning output from the model so far.

type: "summary\_text"

The type of the object. Always `summary_text`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ComputerScreenshotContent object { detail, file\_id, image\_url, type }

A screenshot of a computer.

detail: "low" or "high" or "auto" or "original"

The detail level of the screenshot image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

file\_id: string

The identifier of an uploaded file that contains the screenshot.

image\_url: string

The URL of the screenshot image.

formaturi

type: "computer\_screenshot"

Specifies the event type. For a computer screenshot, this property is always set to `computer_screenshot`.

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

role: "unknown" or "user" or "assistant" or 5 more

The role of the message. One of `unknown`, `user`, `assistant`, `system`, `critic`, `discriminator`, `developer`, or `tool`.

status: "in\_progress" or "completed" or "incomplete"

The status of item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "message"

The type of the message. Always set to `message`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

FunctionCall object { arguments, call\_id, name, 4 more }

A tool call to run a function. See the [function calling guide](https://developers.openai.com/docs/guides/function-calling) for more information.

arguments: string

A JSON string of the arguments to pass to the function.

call\_id: string

The unique ID of the function tool call generated by the model.

name: string

The name of the function to run.

type: "function\_call"

The type of the function tool call. Always `function_call`.

id: optional string

The unique ID of the function tool call.

namespace: optional string

The namespace of the function to run.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

ToolSearchCall object { id, arguments, call\_id, 4 more }

id: string

The unique ID of the tool search call item.

arguments: unknown

Arguments used for the tool search call.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search call item that was recorded.

type: "tool\_search\_call"

The type of the item. Always `tool_search_call`.

created\_by: optional string

The identifier of the actor that created the item.

ToolSearchOutput object { id, call\_id, execution, 4 more }

id: string

The unique ID of the tool search output item.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search output item that was recorded.

tools: array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

The loaded tool definitions returned by tool search.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

type: "tool\_search\_output"

The type of the item. Always `tool_search_output`.

created\_by: optional string

The identifier of the actor that created the item.

FunctionCallOutput object { call\_id, output, type, 2 more }

The output of a function tool call.

call\_id: string

The unique ID of the function tool call generated by the model.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the function call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the function call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the function call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

type: "function\_call\_output"

The type of the function tool call output. Always `function_call_output`.

id: optional string

The unique ID of the function tool call output. Populated when this item is returned via API.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

FileSearchCall object { id, queries, status, 2 more }

The results of a file search tool call. See the [file search guide](https://developers.openai.com/docs/guides/tools-file-search) for more information.

id: string

The unique ID of the file search tool call.

queries: array of string

The queries used to search for files.

status: "in\_progress" or "searching" or "completed" or 2 more

The status of the file search tool call. One of `in_progress`, `searching`, `incomplete` or `failed`,

type: "file\_search\_call"

The type of the file search tool call. Always `file_search_call`.

results: optional array of object { attributes, file\_id, filename, 2 more }

The results of the file search tool call.

attributes: optional map\[string or number or boolean\]

Set of 16 key-value pairs that can be attached to an object. This can be useful for storing additional information about the object in a structured format, and querying for objects via API or the dashboard. Keys are strings with a maximum length of 64 characters. Values are strings with a maximum length of 512 characters, booleans, or numbers.

file\_id: optional string

The unique ID of the file.

filename: optional string

The name of the file.

score: optional number

The relevance score of the file - a value between 0 and 1.

formatfloat

text: optional string

The text that was retrieved from the file.

WebSearchCall object { id, action, status, type }

The results of a web search tool call. See the [web search guide](https://developers.openai.com/docs/guides/tools-web-search) for more information.

id: string

The unique ID of the web search tool call.

action: object { query, type, queries, sources } or object { type, url } or object { pattern, type, url }

An object describing the specific action taken in this web search call. Includes details on how the model used the web (search, open\_page, find\_in\_page).

One of the following:

Search object { query, type, queries, sources }

Action type “search” - Performs a web search query.

query: string

\[DEPRECATED\] The search query.

type: "search"

The action type.

queries: optional array of string

The search queries.

sources: optional array of object { type, url }

The sources used in the search.

type: "url"

The type of source. Always `url`.

url: string

The URL of the source.

formaturi

OpenPage object { type, url }

Action type “open\_page” - Opens a specific URL from search results.

type: "open\_page"

The action type.

url: optional string

The URL opened by the model.

formaturi

FindInPage object { pattern, type, url }

Action type “find\_in\_page”: Searches for a pattern within a loaded page.

pattern: string

The pattern or text to search for within the page.

type: "find\_in\_page"

The action type.

url: string

The URL of the page searched for the pattern.

formaturi

status: "in\_progress" or "searching" or "completed" or "failed"

The status of the web search tool call.

type: "web\_search\_call"

The type of the web search tool call. Always `web_search_call`.

ImageGenerationCall object { id, result, status, type }

An image generation request made by the model.

id: string

The unique ID of the image generation call.

result: string

The generated image encoded in base64.

status: "in\_progress" or "completed" or "generating" or "failed"

The status of the image generation call.

type: "image\_generation\_call"

The type of the image generation call. Always `image_generation_call`.

ComputerCall object { id, call\_id, pending\_safety\_checks, 4 more }

A tool call to a computer use tool. See the [computer use guide](https://developers.openai.com/docs/guides/tools-computer-use) for more information.

id: string

The unique ID of the computer call.

call\_id: string

An identifier used when responding to the tool call with output.

pending\_safety\_checks: array of object { id, code, message }

The pending safety checks for the computer call.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "computer\_call"

The type of the computer call. Always `computer_call`.

action: optional [ComputerAction](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action%20%3E%20\(schema\))

A click action.

actions: optional [ComputerActionList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action_list%20%3E%20\(schema\)) { Click, DoubleClick, Drag, 6 more }

Flattened batched actions for `computer_use`. Each action includes an `type` discriminator and action-specific fields.

ComputerCallOutput object { id, call\_id, output, 4 more }

id: string

The unique ID of the computer call tool output.

call\_id: string

The ID of the computer tool call that produced the output.

output: [ResponseComputerToolCallOutputScreenshot](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_computer_tool_call_output_screenshot%20%3E%20\(schema\)) { type, file\_id, image\_url }

A computer screenshot image used with the computer use tool.

status: "completed" or "incomplete" or "failed" or "in\_progress"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "computer\_call\_output"

The type of the computer tool call output. Always `computer_call_output`.

acknowledged\_safety\_checks: optional array of object { id, code, message }

The safety checks reported by the API that have been acknowledged by the developer.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

created\_by: optional string

The identifier of the actor that created the item.

Reasoning object { id, summary, type, 3 more }

A description of the chain of thought used by a reasoning model while generating a response. Be sure to include these items in your `input` to the Responses API for subsequent turns of a conversation if you are manually [managing context](https://developers.openai.com/docs/guides/conversation-state).

id: string

The unique identifier of the reasoning content.

summary: array of [SummaryTextContent](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20summary_text_content%20%3E%20\(schema\)) { text, type }

Reasoning summary content.

text: string

A summary of the reasoning output from the model so far.

type: "summary\_text"

The type of the object. Always `summary_text`.

type: "reasoning"

The type of the object. Always `reasoning`.

content: optional array of object { text, type }

Reasoning text content.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

encrypted\_content: optional string

The encrypted content of the reasoning item - populated when a response is generated with `reasoning.encrypted_content` in the `include` parameter.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

Compaction object { id, encrypted\_content, type, created\_by }

id: string

The unique ID of the compaction item.

encrypted\_content: string

The encrypted content that was produced by compaction.

type: "compaction"

The type of the item. Always `compaction`.

created\_by: optional string

The identifier of the actor that created the item.

CodeInterpreterCall object { id, code, container\_id, 3 more }

A tool call to run code.

id: string

The unique ID of the code interpreter tool call.

code: string

The code to run, or null if not available.

container\_id: string

The ID of the container used to run the code.

outputs: array of object { logs, type } or object { type, url }

The outputs generated by the code interpreter, such as logs or images. Can be null if no outputs are available.

One of the following:

Logs object { logs, type }

The logs output from the code interpreter.

logs: string

The logs output from the code interpreter.

type: "logs"

The type of the output. Always `logs`.

Image object { type, url }

The image output from the code interpreter.

type: "image"

The type of the output. Always `image`.

url: string

The URL of the image output from the code interpreter.

formaturi

status: "in\_progress" or "completed" or "incomplete" or 2 more

The status of the code interpreter tool call. Valid values are `in_progress`, `completed`, `incomplete`, `interpreting`, and `failed`.

type: "code\_interpreter\_call"

The type of the code interpreter tool call. Always `code_interpreter_call`.

LocalShellCall object { id, action, call\_id, 2 more }

A tool call to run a command on the local shell.

id: string

The unique ID of the local shell call.

action: object { command, env, type, 3 more }

Execute a shell command on the server.

command: array of string

The command to run.

env: map\[string\]

Environment variables to set for the command.

type: "exec"

The type of the local shell action. Always `exec`.

timeout\_ms: optional number

Optional timeout in milliseconds for the command.

user: optional string

Optional user to run the command as.

working\_directory: optional string

Optional working directory to run the command in.

call\_id: string

The unique ID of the local shell tool call generated by the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the local shell call.

type: "local\_shell\_call"

The type of the local shell call. Always `local_shell_call`.

LocalShellCallOutput object { id, output, type, status }

The output of a local shell tool call.

id: string

The unique ID of the local shell tool call generated by the model.

output: string

A JSON string of the output of the local shell tool call.

type: "local\_shell\_call\_output"

The type of the local shell tool call output. Always `local_shell_call_output`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`.

ShellCall object { id, action, call\_id, 4 more }

A tool call that executes one or more shell commands in a managed environment.

id: string

The unique ID of the shell tool call. Populated when this item is returned via API.

action: object { commands, max\_output\_length, timeout\_ms }

The shell commands and limits that describe how to run the tool call.

commands: array of string

max\_output\_length: number

Optional maximum number of characters to return from each command.

timeout\_ms: number

Optional timeout in milliseconds for the commands.

call\_id: string

The unique ID of the shell tool call generated by the model.

environment: [ResponseLocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_local_environment%20%3E%20\(schema\)) { type } or [ResponseContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_container_reference%20%3E%20\(schema\)) { container\_id, type }

Represents the use of a local environment to perform shell actions.

One of the following:

ResponseLocalEnvironment object { type }

Represents the use of a local environment to perform shell actions.

type: "local"

The environment type. Always `local`.

ResponseContainerReference object { container\_id, type }

Represents a container created with /v1/containers.

container\_id: string

type: "container\_reference"

The environment type. Always `container_reference`.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call"

The type of the item. Always `shell_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ShellCallOutput object { id, call\_id, max\_output\_length, 4 more }

The output of a shell tool call that was emitted.

id: string

The unique ID of the shell call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the shell tool call generated by the model.

max\_output\_length: number

The maximum length of the shell command output. This is generated by the model and should be passed back with the raw output.

output: array of object { outcome, stderr, stdout, created\_by }

An array of shell call output contents

outcome: object { type } or object { exit\_code, type }

Represents either an exit outcome (with an exit code) or a timeout outcome for a shell call output chunk.

One of the following:

Timeout object { type }

Indicates that the shell call exceeded its configured time limit.

type: "timeout"

The outcome type. Always `timeout`.

Exit object { exit\_code, type }

Indicates that the shell commands finished and returned an exit code.

exit\_code: number

Exit code from the shell process.

type: "exit"

The outcome type. Always `exit`.

stderr: string

The standard error output that was captured.

stdout: string

The standard output that was captured.

created\_by: optional string

The identifier of the actor that created the item.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call output. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call\_output"

The type of the shell call output. Always `shell_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

ApplyPatchCall object { id, call\_id, operation, 3 more }

A tool call that applies file diffs by creating, deleting, or updating files.

id: string

The unique ID of the apply patch tool call. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

operation: object { diff, path, type } or object { path, type } or object { diff, path, type }

One of the create\_file, delete\_file, or update\_file operations applied via apply\_patch.

One of the following:

CreateFile object { diff, path, type }

Instruction describing how to create a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to create.

type: "create\_file"

Create a new file with the provided diff.

DeleteFile object { path, type }

Instruction describing how to delete a file via the apply\_patch tool.

path: string

Path of the file to delete.

type: "delete\_file"

Delete the specified file.

UpdateFile object { diff, path, type }

Instruction describing how to update a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to update.

type: "update\_file"

Update an existing file with the provided diff.

status: "in\_progress" or "completed"

The status of the apply patch tool call. One of `in_progress` or `completed`.

One of the following:

"in\_progress"

"completed"

type: "apply\_patch\_call"

The type of the item. Always `apply_patch_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ApplyPatchCallOutput object { id, call\_id, status, 3 more }

The output emitted by an apply patch tool call.

id: string

The unique ID of the apply patch tool call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

status: "completed" or "failed"

The status of the apply patch tool call output. One of `completed` or `failed`.

One of the following:

"completed"

"failed"

type: "apply\_patch\_call\_output"

The type of the item. Always `apply_patch_call_output`.

created\_by: optional string

The ID of the entity that created this tool call output.

output: optional string

Optional textual output returned by the apply patch tool.

McpListTools object { id, server\_label, tools, 2 more }

A list of tools available on an MCP server.

id: string

The unique ID of the list.

server\_label: string

The label of the MCP server.

tools: array of object { input\_schema, name, annotations, description }

The tools available on the server.

input\_schema: unknown

The JSON schema describing the tool’s input.

name: string

The name of the tool.

annotations: optional unknown

Additional annotations about the tool.

description: optional string

The description of the tool.

type: "mcp\_list\_tools"

The type of the item. Always `mcp_list_tools`.

error: optional string

Error message if the server could not list tools.

McpApprovalRequest object { id, arguments, name, 2 more }

A request for human approval of a tool invocation.

id: string

The unique ID of the approval request.

arguments: string

A JSON string of arguments for the tool.

name: string

The name of the tool to run.

server\_label: string

The label of the MCP server making the request.

type: "mcp\_approval\_request"

The type of the item. Always `mcp_approval_request`.

McpApprovalResponse object { id, approval\_request\_id, approve, 2 more }

A response to an MCP approval request.

id: string

The unique ID of the approval response

approval\_request\_id: string

The ID of the approval request being answered.

approve: boolean

Whether the request was approved.

type: "mcp\_approval\_response"

The type of the item. Always `mcp_approval_response`.

reason: optional string

Optional reason for the decision.

McpCall object { id, arguments, name, 6 more }

An invocation of a tool on an MCP server.

id: string

The unique ID of the tool call.

arguments: string

A JSON string of the arguments passed to the tool.

name: string

The name of the tool that was run.

server\_label: string

The label of the MCP server running the tool.

type: "mcp\_call"

The type of the item. Always `mcp_call`.

approval\_request\_id: optional string

Unique identifier for the MCP tool call approval request. Include this value in a subsequent `mcp_approval_response` input to approve or reject the corresponding tool call.

error: optional string

The error from the tool call, if any.

output: optional string

The output from the tool call.

status: optional "in\_progress" or "completed" or "incomplete" or 2 more

The status of the tool call. One of `in_progress`, `completed`, `incomplete`, `calling`, or `failed`.

CustomToolCall object { call\_id, input, name, 3 more }

A call to a custom tool created by the model.

call\_id: string

An identifier used to map this custom tool call to a tool call output.

input: string

The input for the custom tool call generated by the model.

name: string

The name of the custom tool being called.

type: "custom\_tool\_call"

The type of the custom tool call. Always `custom_tool_call`.

id: optional string

The unique ID of the custom tool call in the OpenAI platform.

namespace: optional string

The namespace of the custom tool being called.

CustomToolCallOutput object { call\_id, output, type, id }

The output of a custom tool call from your code, being sent back to the model.

call\_id: string

The call ID, used to map this custom tool call output to a custom tool call.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the custom tool call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the custom tool call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the custom tool call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

type: "custom\_tool\_call\_output"

The type of the custom tool call output. Always `custom_tool_call_output`.

id: optional string

The unique ID of the custom tool call output in the OpenAI platform.

usage: [ResponseUsage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_usage%20%3E%20\(schema\)) { input\_tokens, input\_tokens\_details, output\_tokens, 2 more }

Token accounting for the compaction pass, including cached, reasoning, and total tokens.

ComputerAction = object { button, type, x, 2 more } or object { keys, type, x, y } or object { path, type, keys } or 6 more

A click action.

One of the following:

Click object { button, type, x, 2 more }

A click action.

button: "left" or "right" or "wheel" or 2 more

Indicates which mouse button was pressed during the click. One of `left`, `right`, `wheel`, `back`, or `forward`.

type: "click"

Specifies the event type. For a click action, this property is always `click`.

x: number

The x-coordinate where the click occurred.

y: number

The y-coordinate where the click occurred.

keys: optional array of string

The keys being held while clicking.

DoubleClick object { keys, type, x, y }

A double click action.

keys: array of string

The keys being held while double-clicking.

type: "double\_click"

Specifies the event type. For a double click action, this property is always set to `double_click`.

x: number

The x-coordinate where the double click occurred.

y: number

The y-coordinate where the double click occurred.

Drag object { path, type, keys }

A drag action.

path: array of object { x, y }

An array of coordinates representing the path of the drag action. Coordinates will appear as an array of objects, eg

```plaintext
[
  { x: 100, y: 200 },
  { x: 200, y: 300 }
]
```

x: number

The x-coordinate.

y: number

The y-coordinate.

type: "drag"

Specifies the event type. For a drag action, this property is always set to `drag`.

keys: optional array of string

The keys being held while dragging the mouse.

Keypress object { keys, type }

A collection of keypresses the model would like to perform.

keys: array of string

The combination of keys the model is requesting to be pressed. This is an array of strings, each representing a key.

type: "keypress"

Specifies the event type. For a keypress action, this property is always set to `keypress`.

Move object { type, x, y, keys }

A mouse move action.

type: "move"

Specifies the event type. For a move action, this property is always set to `move`.

x: number

The x-coordinate to move to.

y: number

The y-coordinate to move to.

keys: optional array of string

The keys being held while moving the mouse.

Screenshot object { type }

A screenshot action.

type: "screenshot"

Specifies the event type. For a screenshot action, this property is always set to `screenshot`.

Scroll object { scroll\_x, scroll\_y, type, 3 more }

A scroll action.

scroll\_x: number

The horizontal scroll distance.

scroll\_y: number

The vertical scroll distance.

type: "scroll"

Specifies the event type. For a scroll action, this property is always set to `scroll`.

x: number

The x-coordinate where the scroll occurred.

y: number

The y-coordinate where the scroll occurred.

keys: optional array of string

The keys being held while scrolling.

Type object { text, type }

An action to type in text.

text: string

The text to type.

type: "type"

Specifies the event type. For a type action, this property is always set to `type`.

Wait object { type }

A wait action.

type: "wait"

Specifies the event type. For a wait action, this property is always set to `wait`.

ComputerActionList = array of [ComputerAction](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action%20%3E%20\(schema\))

Flattened batched actions for `computer_use`. Each action includes an `type` discriminator and action-specific fields.

One of the following:

Click object { button, type, x, 2 more }

A click action.

button: "left" or "right" or "wheel" or 2 more

Indicates which mouse button was pressed during the click. One of `left`, `right`, `wheel`, `back`, or `forward`.

type: "click"

Specifies the event type. For a click action, this property is always `click`.

x: number

The x-coordinate where the click occurred.

y: number

The y-coordinate where the click occurred.

keys: optional array of string

The keys being held while clicking.

DoubleClick object { keys, type, x, y }

A double click action.

keys: array of string

The keys being held while double-clicking.

type: "double\_click"

Specifies the event type. For a double click action, this property is always set to `double_click`.

x: number

The x-coordinate where the double click occurred.

y: number

The y-coordinate where the double click occurred.

Drag object { path, type, keys }

A drag action.

path: array of object { x, y }

An array of coordinates representing the path of the drag action. Coordinates will appear as an array of objects, eg

```plaintext
[
  { x: 100, y: 200 },
  { x: 200, y: 300 }
]
```

x: number

The x-coordinate.

y: number

The y-coordinate.

type: "drag"

Specifies the event type. For a drag action, this property is always set to `drag`.

keys: optional array of string

The keys being held while dragging the mouse.

Keypress object { keys, type }

A collection of keypresses the model would like to perform.

keys: array of string

The combination of keys the model is requesting to be pressed. This is an array of strings, each representing a key.

type: "keypress"

Specifies the event type. For a keypress action, this property is always set to `keypress`.

Move object { type, x, y, keys }

A mouse move action.

type: "move"

Specifies the event type. For a move action, this property is always set to `move`.

x: number

The x-coordinate to move to.

y: number

The y-coordinate to move to.

keys: optional array of string

The keys being held while moving the mouse.

Screenshot object { type }

A screenshot action.

type: "screenshot"

Specifies the event type. For a screenshot action, this property is always set to `screenshot`.

Scroll object { scroll\_x, scroll\_y, type, 3 more }

A scroll action.

scroll\_x: number

The horizontal scroll distance.

scroll\_y: number

The vertical scroll distance.

type: "scroll"

Specifies the event type. For a scroll action, this property is always set to `scroll`.

x: number

The x-coordinate where the scroll occurred.

y: number

The y-coordinate where the scroll occurred.

keys: optional array of string

The keys being held while scrolling.

Type object { text, type }

An action to type in text.

text: string

The text to type.

type: "type"

Specifies the event type. For a type action, this property is always set to `type`.

Wait object { type }

A wait action.

type: "wait"

Specifies the event type. For a wait action, this property is always set to `wait`.

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyDomainSecret object { domain, name, value }

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

EasyInputMessage object { content, role, phase, type }

A message input to the model with a role indicating instruction following hierarchy. Instructions given with the `developer` or `system` role take precedence over instructions given with the `user` role. Messages with the `assistant` role are presumed to have been generated by the model in previous interactions.

content: string or [ResponseInputMessageContentList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_content_list%20%3E%20\(schema\)) {,, }

Text, image, or audio input to the model, used to generate a response. Can also contain previous assistant responses.

One of the following:

TextInput = string

A text input to the model.

ResponseInputMessageContentList = array of [ResponseInputContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_content%20%3E%20\(schema\))

A list of one or many input items to the model, containing different content types.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

role: "user" or "assistant" or "system" or "developer"

The role of the message input. One of `user`, `assistant`, `system`, or `developer`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

type: optional "message"

The type of the message input. Always `message`.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

InlineSkillSource object { data, media\_type, type }

Inline skill payload

data: string

Base64-encoded skill zip bundle.

maxLength70254592

minLength1

media\_type: "application/zip"

The media type of the inline skill payload. Must be `application/zip`.

type: "base64"

The type of the inline skill source. Must be `base64`.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

LocalSkill object { description, name, path }

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

Response object { id, created\_at, error, 30 more }

id: string

Unique identifier for this Response.

created\_at: number

Unix timestamp (in seconds) of when this Response was created.

formatunixtime

error: [ResponseError](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_error%20%3E%20\(schema\)) { code, message }

An error object returned when the model fails to generate a Response.

incomplete\_details: object { reason }

Details about why the response is incomplete.

reason: optional "max\_output\_tokens" or "content\_filter"

The reason why the response is incomplete.

One of the following:

"max\_output\_tokens"

"content\_filter"

instructions: string or array of [EasyInputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20easy_input_message%20%3E%20\(schema\)) { content, role, phase, type } or object { content, role, status, type } or [ResponseOutputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_message%20%3E%20\(schema\)) { id, content, role, 3 more } or 25 more

A system (or developer) message inserted into the model’s context.

When using along with `previous_response_id`, the instructions from a previous response will not be carried over to the next response. This makes it simple to swap out system (or developer) messages in new responses.

One of the following:

string

A text input to the model, equivalent to a text input with the `developer` role.

InputItemList = array of [EasyInputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20easy_input_message%20%3E%20\(schema\)) { content, role, phase, type } or object { content, role, status, type } or [ResponseOutputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_message%20%3E%20\(schema\)) { id, content, role, 3 more } or 25 more

A list of one or many input items to the model, containing different content types.

One of the following:

EasyInputMessage object { content, role, phase, type }

A message input to the model with a role indicating instruction following hierarchy. Instructions given with the `developer` or `system` role take precedence over instructions given with the `user` role. Messages with the `assistant` role are presumed to have been generated by the model in previous interactions.

content: string or [ResponseInputMessageContentList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_content_list%20%3E%20\(schema\)) {,, }

Text, image, or audio input to the model, used to generate a response. Can also contain previous assistant responses.

One of the following:

TextInput = string

A text input to the model.

ResponseInputMessageContentList = array of [ResponseInputContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_content%20%3E%20\(schema\))

A list of one or many input items to the model, containing different content types.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

role: "user" or "assistant" or "system" or "developer"

The role of the message input. One of `user`, `assistant`, `system`, or `developer`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

type: optional "message"

The type of the message input. Always `message`.

Message object { content, role, status, type }

A message input to the model with a role indicating instruction following hierarchy. Instructions given with the `developer` or `system` role take precedence over instructions given with the `user` role.

content: [ResponseInputMessageContentList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_content_list%20%3E%20\(schema\)) {,, }

A list of one or many input items to the model, containing different content types.

role: "user" or "system" or "developer"

The role of the message input. One of `user`, `system`, or `developer`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: optional "message"

The type of the message input. Always set to `message`.

ResponseOutputMessage object { id, content, role, 3 more }

An output message from the model.

id: string

The unique ID of the output message.

content: array of [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type }

The content of the output message.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

role: "assistant"

The role of the output message. Always `assistant`.

status: "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "message"

The type of the output message. Always `message`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

FileSearchCall object { id, queries, status, 2 more }

The results of a file search tool call. See the [file search guide](https://developers.openai.com/docs/guides/tools-file-search) for more information.

id: string

The unique ID of the file search tool call.

queries: array of string

The queries used to search for files.

status: "in\_progress" or "searching" or "completed" or 2 more

The status of the file search tool call. One of `in_progress`, `searching`, `incomplete` or `failed`,

type: "file\_search\_call"

The type of the file search tool call. Always `file_search_call`.

results: optional array of object { attributes, file\_id, filename, 2 more }

The results of the file search tool call.

attributes: optional map\[string or number or boolean\]

Set of 16 key-value pairs that can be attached to an object. This can be useful for storing additional information about the object in a structured format, and querying for objects via API or the dashboard. Keys are strings with a maximum length of 64 characters. Values are strings with a maximum length of 512 characters, booleans, or numbers.

file\_id: optional string

The unique ID of the file.

filename: optional string

The name of the file.

score: optional number

The relevance score of the file - a value between 0 and 1.

formatfloat

text: optional string

The text that was retrieved from the file.

ComputerCall object { id, call\_id, pending\_safety\_checks, 4 more }

A tool call to a computer use tool. See the [computer use guide](https://developers.openai.com/docs/guides/tools-computer-use) for more information.

id: string

The unique ID of the computer call.

call\_id: string

An identifier used when responding to the tool call with output.

pending\_safety\_checks: array of object { id, code, message }

The pending safety checks for the computer call.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "computer\_call"

The type of the computer call. Always `computer_call`.

action: optional [ComputerAction](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action%20%3E%20\(schema\))

A click action.

actions: optional [ComputerActionList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action_list%20%3E%20\(schema\)) { Click, DoubleClick, Drag, 6 more }

Flattened batched actions for `computer_use`. Each action includes an `type` discriminator and action-specific fields.

ComputerCallOutput object { call\_id, output, type, 3 more }

The output of a computer tool call.

call\_id: string

The ID of the computer tool call that produced the output.

maxLength64

minLength1

output: [ResponseComputerToolCallOutputScreenshot](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_computer_tool_call_output_screenshot%20%3E%20\(schema\)) { type, file\_id, image\_url }

A computer screenshot image used with the computer use tool.

type: "computer\_call\_output"

The type of the computer tool call output. Always `computer_call_output`.

id: optional string

The ID of the computer tool call output.

acknowledged\_safety\_checks: optional array of object { id, code, message }

The safety checks reported by the API that have been acknowledged by the developer.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

WebSearchCall object { id, action, status, type }

The results of a web search tool call. See the [web search guide](https://developers.openai.com/docs/guides/tools-web-search) for more information.

id: string

The unique ID of the web search tool call.

action: object { query, type, queries, sources } or object { type, url } or object { pattern, type, url }

An object describing the specific action taken in this web search call. Includes details on how the model used the web (search, open\_page, find\_in\_page).

One of the following:

Search object { query, type, queries, sources }

Action type “search” - Performs a web search query.

query: string

\[DEPRECATED\] The search query.

type: "search"

The action type.

queries: optional array of string

The search queries.

sources: optional array of object { type, url }

The sources used in the search.

type: "url"

The type of source. Always `url`.

url: string

The URL of the source.

formaturi

OpenPage object { type, url }

Action type “open\_page” - Opens a specific URL from search results.

type: "open\_page"

The action type.

url: optional string

The URL opened by the model.

formaturi

FindInPage object { pattern, type, url }

Action type “find\_in\_page”: Searches for a pattern within a loaded page.

pattern: string

The pattern or text to search for within the page.

type: "find\_in\_page"

The action type.

url: string

The URL of the page searched for the pattern.

formaturi

status: "in\_progress" or "searching" or "completed" or "failed"

The status of the web search tool call.

type: "web\_search\_call"

The type of the web search tool call. Always `web_search_call`.

FunctionCall object { arguments, call\_id, name, 4 more }

A tool call to run a function. See the [function calling guide](https://developers.openai.com/docs/guides/function-calling) for more information.

arguments: string

A JSON string of the arguments to pass to the function.

call\_id: string

The unique ID of the function tool call generated by the model.

name: string

The name of the function to run.

type: "function\_call"

The type of the function tool call. Always `function_call`.

id: optional string

The unique ID of the function tool call.

namespace: optional string

The namespace of the function to run.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

FunctionCallOutput object { call\_id, output, type, 2 more }

The output of a function tool call.

call\_id: string

The unique ID of the function tool call generated by the model.

maxLength64

minLength1

output: string or array of [ResponseInputTextContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text_content%20%3E%20\(schema\)) { text, type } or [ResponseInputImageContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image_content%20%3E%20\(schema\)) { type, detail, file\_id, image\_url } or [ResponseInputFileContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file_content%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the function tool call.

One of the following:

string

A JSON string of the output of the function tool call.

array of [ResponseInputTextContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text_content%20%3E%20\(schema\)) { text, type } or [ResponseInputImageContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image_content%20%3E%20\(schema\)) { type, detail, file\_id, image\_url } or [ResponseInputFileContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file_content%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

An array of content outputs (text, image, file) for the function tool call.

One of the following:

ResponseInputTextContent object { text, type }

A text input to the model.

text: string

The text input to the model.

maxLength10485760

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImageContent object { type, detail, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision)

type: "input\_image"

The type of the input item. Always `input_image`.

detail: optional "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

maxLength20971520

formaturi

ResponseInputFileContent object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The base64-encoded data of the file to be sent to the model.

maxLength73400320

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

type: "function\_call\_output"

The type of the function tool call output. Always `function_call_output`.

id: optional string

The unique ID of the function tool call output. Populated when this item is returned via API.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

ToolSearchCall object { arguments, type, id, 3 more }

arguments: unknown

The arguments supplied to the tool search call.

type: "tool\_search\_call"

The item type. Always `tool_search_call`.

id: optional string

The unique ID of this tool search call.

call\_id: optional string

The unique ID of the tool search call generated by the model.

maxLength64

minLength1

execution: optional "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: optional "in\_progress" or "completed" or "incomplete"

The status of the tool search call.

ToolSearchOutput object { tools, type, id, 3 more }

tools: array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

The loaded tool definitions returned by the tool search output.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

type: "tool\_search\_output"

The item type. Always `tool_search_output`.

id: optional string

The unique ID of this tool search output.

call\_id: optional string

The unique ID of the tool search call generated by the model.

maxLength64

minLength1

execution: optional "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: optional "in\_progress" or "completed" or "incomplete"

The status of the tool search output.

Reasoning object { id, summary, type, 3 more }

A description of the chain of thought used by a reasoning model while generating a response. Be sure to include these items in your `input` to the Responses API for subsequent turns of a conversation if you are manually [managing context](https://developers.openai.com/docs/guides/conversation-state).

id: string

The unique identifier of the reasoning content.

summary: array of [SummaryTextContent](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20summary_text_content%20%3E%20\(schema\)) { text, type }

Reasoning summary content.

text: string

A summary of the reasoning output from the model so far.

type: "summary\_text"

The type of the object. Always `summary_text`.

type: "reasoning"

The type of the object. Always `reasoning`.

content: optional array of object { text, type }

Reasoning text content.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

encrypted\_content: optional string

The encrypted content of the reasoning item - populated when a response is generated with `reasoning.encrypted_content` in the `include` parameter.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

Compaction object { encrypted\_content, type, id }

encrypted\_content: string

The encrypted content of the compaction summary.

maxLength10485760

type: "compaction"

The type of the item. Always `compaction`.

id: optional string

The ID of the compaction item.

ImageGenerationCall object { id, result, status, type }

An image generation request made by the model.

id: string

The unique ID of the image generation call.

result: string

The generated image encoded in base64.

status: "in\_progress" or "completed" or "generating" or "failed"

The status of the image generation call.

type: "image\_generation\_call"

The type of the image generation call. Always `image_generation_call`.

CodeInterpreterCall object { id, code, container\_id, 3 more }

A tool call to run code.

id: string

The unique ID of the code interpreter tool call.

code: string

The code to run, or null if not available.

container\_id: string

The ID of the container used to run the code.

outputs: array of object { logs, type } or object { type, url }

The outputs generated by the code interpreter, such as logs or images. Can be null if no outputs are available.

One of the following:

Logs object { logs, type }

The logs output from the code interpreter.

logs: string

The logs output from the code interpreter.

type: "logs"

The type of the output. Always `logs`.

Image object { type, url }

The image output from the code interpreter.

type: "image"

The type of the output. Always `image`.

url: string

The URL of the image output from the code interpreter.

formaturi

status: "in\_progress" or "completed" or "incomplete" or 2 more

The status of the code interpreter tool call. Valid values are `in_progress`, `completed`, `incomplete`, `interpreting`, and `failed`.

type: "code\_interpreter\_call"

The type of the code interpreter tool call. Always `code_interpreter_call`.

LocalShellCall object { id, action, call\_id, 2 more }

A tool call to run a command on the local shell.

id: string

The unique ID of the local shell call.

action: object { command, env, type, 3 more }

Execute a shell command on the server.

command: array of string

The command to run.

env: map\[string\]

Environment variables to set for the command.

type: "exec"

The type of the local shell action. Always `exec`.

timeout\_ms: optional number

Optional timeout in milliseconds for the command.

user: optional string

Optional user to run the command as.

working\_directory: optional string

Optional working directory to run the command in.

call\_id: string

The unique ID of the local shell tool call generated by the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the local shell call.

type: "local\_shell\_call"

The type of the local shell call. Always `local_shell_call`.

LocalShellCallOutput object { id, output, type, status }

The output of a local shell tool call.

id: string

The unique ID of the local shell tool call generated by the model.

output: string

A JSON string of the output of the local shell tool call.

type: "local\_shell\_call\_output"

The type of the local shell tool call output. Always `local_shell_call_output`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`.

ShellCall object { action, call\_id, type, 3 more }

A tool representing a request to execute one or more shell commands.

action: object { commands, max\_output\_length, timeout\_ms }

The shell commands and limits that describe how to run the tool call.

commands: array of string

Ordered shell commands for the execution environment to run.

max\_output\_length: optional number

Maximum number of UTF-8 characters to capture from combined stdout and stderr output.

timeout\_ms: optional number

Maximum wall-clock time in milliseconds to allow the shell commands to run.

call\_id: string

The unique ID of the shell tool call generated by the model.

maxLength64

minLength1

type: "shell\_call"

The type of the item. Always `shell_call`.

id: optional string

The unique ID of the shell tool call. Populated when this item is returned via API.

environment: optional [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

The environment to execute the shell commands in.

One of the following:

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

status: optional "in\_progress" or "completed" or "incomplete"

The status of the shell call. One of `in_progress`, `completed`, or `incomplete`.

ShellCallOutput object { call\_id, output, type, 3 more }

The streamed output items emitted by a shell tool call.

call\_id: string

The unique ID of the shell tool call generated by the model.

maxLength64

minLength1

output: array of [ResponseFunctionShellCallOutputContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_function_shell_call_output_content%20%3E%20\(schema\)) { outcome, stderr, stdout }

Captured chunks of stdout and stderr output, along with their associated outcomes.

outcome: object { type } or object { exit\_code, type }

The exit or timeout outcome associated with this shell call.

One of the following:

Timeout object { type }

Indicates that the shell call exceeded its configured time limit.

type: "timeout"

The outcome type. Always `timeout`.

Exit object { exit\_code, type }

Indicates that the shell commands finished and returned an exit code.

exit\_code: number

The exit code returned by the shell process.

type: "exit"

The outcome type. Always `exit`.

stderr: string

Captured stderr output for the shell call.

maxLength10485760

stdout: string

Captured stdout output for the shell call.

maxLength10485760

type: "shell\_call\_output"

The type of the item. Always `shell_call_output`.

id: optional string

The unique ID of the shell tool call output. Populated when this item is returned via API.

max\_output\_length: optional number

The maximum number of UTF-8 characters captured for this shell call’s combined output.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the shell call output.

ApplyPatchCall object { call\_id, operation, status, 2 more }

A tool call representing a request to create, delete, or update files using diff patches.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

maxLength64

minLength1

operation: object { diff, path, type } or object { path, type } or object { diff, path, type }

The specific create, delete, or update instruction for the apply\_patch tool call.

One of the following:

CreateFile object { diff, path, type }

Instruction for creating a new file via the apply\_patch tool.

diff: string

Unified diff content to apply when creating the file.

maxLength10485760

path: string

Path of the file to create relative to the workspace root.

minLength1

type: "create\_file"

The operation type. Always `create_file`.

DeleteFile object { path, type }

Instruction for deleting an existing file via the apply\_patch tool.

path: string

Path of the file to delete relative to the workspace root.

minLength1

type: "delete\_file"

The operation type. Always `delete_file`.

UpdateFile object { diff, path, type }

Instruction for updating an existing file via the apply\_patch tool.

diff: string

Unified diff content to apply to the existing file.

maxLength10485760

path: string

Path of the file to update relative to the workspace root.

minLength1

type: "update\_file"

The operation type. Always `update_file`.

status: "in\_progress" or "completed"

The status of the apply patch tool call. One of `in_progress` or `completed`.

One of the following:

"in\_progress"

"completed"

type: "apply\_patch\_call"

The type of the item. Always `apply_patch_call`.

id: optional string

The unique ID of the apply patch tool call. Populated when this item is returned via API.

ApplyPatchCallOutput object { call\_id, status, type, 2 more }

The streamed output emitted by an apply patch tool call.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

maxLength64

minLength1

status: "completed" or "failed"

The status of the apply patch tool call output. One of `completed` or `failed`.

One of the following:

"completed"

"failed"

type: "apply\_patch\_call\_output"

The type of the item. Always `apply_patch_call_output`.

id: optional string

The unique ID of the apply patch tool call output. Populated when this item is returned via API.

output: optional string

Optional human-readable log text from the apply patch tool (e.g., patch results or errors).

maxLength10485760

McpListTools object { id, server\_label, tools, 2 more }

A list of tools available on an MCP server.

id: string

The unique ID of the list.

server\_label: string

The label of the MCP server.

tools: array of object { input\_schema, name, annotations, description }

The tools available on the server.

input\_schema: unknown

The JSON schema describing the tool’s input.

name: string

The name of the tool.

annotations: optional unknown

Additional annotations about the tool.

description: optional string

The description of the tool.

type: "mcp\_list\_tools"

The type of the item. Always `mcp_list_tools`.

error: optional string

Error message if the server could not list tools.

McpApprovalRequest object { id, arguments, name, 2 more }

A request for human approval of a tool invocation.

id: string

The unique ID of the approval request.

arguments: string

A JSON string of arguments for the tool.

name: string

The name of the tool to run.

server\_label: string

The label of the MCP server making the request.

type: "mcp\_approval\_request"

The type of the item. Always `mcp_approval_request`.

McpApprovalResponse object { approval\_request\_id, approve, type, 2 more }

A response to an MCP approval request.

approval\_request\_id: string

The ID of the approval request being answered.

approve: boolean

Whether the request was approved.

type: "mcp\_approval\_response"

The type of the item. Always `mcp_approval_response`.

id: optional string

The unique ID of the approval response

reason: optional string

Optional reason for the decision.

McpCall object { id, arguments, name, 6 more }

An invocation of a tool on an MCP server.

id: string

The unique ID of the tool call.

arguments: string

A JSON string of the arguments passed to the tool.

name: string

The name of the tool that was run.

server\_label: string

The label of the MCP server running the tool.

type: "mcp\_call"

The type of the item. Always `mcp_call`.

approval\_request\_id: optional string

Unique identifier for the MCP tool call approval request. Include this value in a subsequent `mcp_approval_response` input to approve or reject the corresponding tool call.

error: optional string

The error from the tool call, if any.

output: optional string

The output from the tool call.

status: optional "in\_progress" or "completed" or "incomplete" or 2 more

The status of the tool call. One of `in_progress`, `completed`, `incomplete`, `calling`, or `failed`.

CustomToolCallOutput object { call\_id, output, type, id }

The output of a custom tool call from your code, being sent back to the model.

call\_id: string

The call ID, used to map this custom tool call output to a custom tool call.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the custom tool call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the custom tool call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the custom tool call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

type: "custom\_tool\_call\_output"

The type of the custom tool call output. Always `custom_tool_call_output`.

id: optional string

The unique ID of the custom tool call output in the OpenAI platform.

CustomToolCall object { call\_id, input, name, 3 more }

A call to a custom tool created by the model.

call\_id: string

An identifier used to map this custom tool call to a tool call output.

input: string

The input for the custom tool call generated by the model.

name: string

The name of the custom tool being called.

type: "custom\_tool\_call"

The type of the custom tool call. Always `custom_tool_call`.

id: optional string

The unique ID of the custom tool call in the OpenAI platform.

namespace: optional string

The namespace of the custom tool being called.

ItemReference object { id, type }

An internal identifier for an item to reference.

id: string

The ID of the item to reference.

type: optional "item\_reference"

The type of item to reference. Always `item_reference`.

model: [ResponsesModel](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20responses_model%20%3E%20\(schema\))

Model ID used to generate the response, like `gpt-4o` or `o3`. OpenAI offers a wide range of models with different capabilities, performance characteristics, and price points. Refer to the [model guide](https://developers.openai.com/docs/models) to browse and compare available models.

object: "response"

The object type of this resource - always set to `response`.

output: array of [ResponseOutputItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_item%20%3E%20\(schema\))

An array of content items generated by the model.

- The length and order of items in the `output` array is dependent on the model’s response.
- Rather than accessing the first item in the `output` array and assuming it’s an `assistant` message with the content generated by the model, you might consider using the `output_text` property where supported in SDKs.

One of the following:

ResponseOutputMessage object { id, content, role, 3 more }

An output message from the model.

id: string

The unique ID of the output message.

content: array of [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type }

The content of the output message.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

role: "assistant"

The role of the output message. Always `assistant`.

status: "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "message"

The type of the output message. Always `message`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

FileSearchCall object { id, queries, status, 2 more }

The results of a file search tool call. See the [file search guide](https://developers.openai.com/docs/guides/tools-file-search) for more information.

id: string

The unique ID of the file search tool call.

queries: array of string

The queries used to search for files.

status: "in\_progress" or "searching" or "completed" or 2 more

The status of the file search tool call. One of `in_progress`, `searching`, `incomplete` or `failed`,

type: "file\_search\_call"

The type of the file search tool call. Always `file_search_call`.

results: optional array of object { attributes, file\_id, filename, 2 more }

The results of the file search tool call.

attributes: optional map\[string or number or boolean\]

Set of 16 key-value pairs that can be attached to an object. This can be useful for storing additional information about the object in a structured format, and querying for objects via API or the dashboard. Keys are strings with a maximum length of 64 characters. Values are strings with a maximum length of 512 characters, booleans, or numbers.

file\_id: optional string

The unique ID of the file.

filename: optional string

The name of the file.

score: optional number

The relevance score of the file - a value between 0 and 1.

formatfloat

text: optional string

The text that was retrieved from the file.

FunctionCall object { arguments, call\_id, name, 4 more }

A tool call to run a function. See the [function calling guide](https://developers.openai.com/docs/guides/function-calling) for more information.

arguments: string

A JSON string of the arguments to pass to the function.

call\_id: string

The unique ID of the function tool call generated by the model.

name: string

The name of the function to run.

type: "function\_call"

The type of the function tool call. Always `function_call`.

id: optional string

The unique ID of the function tool call.

namespace: optional string

The namespace of the function to run.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

FunctionCallOutput object { id, call\_id, output, 3 more }

id: string

The unique ID of the function call tool output.

call\_id: string

The unique ID of the function tool call generated by the model.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the function call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the function call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the function call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "function\_call\_output"

The type of the function tool call output. Always `function_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

WebSearchCall object { id, action, status, type }

The results of a web search tool call. See the [web search guide](https://developers.openai.com/docs/guides/tools-web-search) for more information.

id: string

The unique ID of the web search tool call.

action: object { query, type, queries, sources } or object { type, url } or object { pattern, type, url }

An object describing the specific action taken in this web search call. Includes details on how the model used the web (search, open\_page, find\_in\_page).

One of the following:

Search object { query, type, queries, sources }

Action type “search” - Performs a web search query.

query: string

\[DEPRECATED\] The search query.

type: "search"

The action type.

queries: optional array of string

The search queries.

sources: optional array of object { type, url }

The sources used in the search.

type: "url"

The type of source. Always `url`.

url: string

The URL of the source.

formaturi

OpenPage object { type, url }

Action type “open\_page” - Opens a specific URL from search results.

type: "open\_page"

The action type.

url: optional string

The URL opened by the model.

formaturi

FindInPage object { pattern, type, url }

Action type “find\_in\_page”: Searches for a pattern within a loaded page.

pattern: string

The pattern or text to search for within the page.

type: "find\_in\_page"

The action type.

url: string

The URL of the page searched for the pattern.

formaturi

status: "in\_progress" or "searching" or "completed" or "failed"

The status of the web search tool call.

type: "web\_search\_call"

The type of the web search tool call. Always `web_search_call`.

ComputerCall object { id, call\_id, pending\_safety\_checks, 4 more }

A tool call to a computer use tool. See the [computer use guide](https://developers.openai.com/docs/guides/tools-computer-use) for more information.

id: string

The unique ID of the computer call.

call\_id: string

An identifier used when responding to the tool call with output.

pending\_safety\_checks: array of object { id, code, message }

The pending safety checks for the computer call.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "computer\_call"

The type of the computer call. Always `computer_call`.

action: optional [ComputerAction](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action%20%3E%20\(schema\))

A click action.

actions: optional [ComputerActionList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action_list%20%3E%20\(schema\)) { Click, DoubleClick, Drag, 6 more }

Flattened batched actions for `computer_use`. Each action includes an `type` discriminator and action-specific fields.

ComputerCallOutput object { id, call\_id, output, 4 more }

id: string

The unique ID of the computer call tool output.

call\_id: string

The ID of the computer tool call that produced the output.

output: [ResponseComputerToolCallOutputScreenshot](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_computer_tool_call_output_screenshot%20%3E%20\(schema\)) { type, file\_id, image\_url }

A computer screenshot image used with the computer use tool.

status: "completed" or "incomplete" or "failed" or "in\_progress"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "computer\_call\_output"

The type of the computer tool call output. Always `computer_call_output`.

acknowledged\_safety\_checks: optional array of object { id, code, message }

The safety checks reported by the API that have been acknowledged by the developer.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

created\_by: optional string

The identifier of the actor that created the item.

Reasoning object { id, summary, type, 3 more }

A description of the chain of thought used by a reasoning model while generating a response. Be sure to include these items in your `input` to the Responses API for subsequent turns of a conversation if you are manually [managing context](https://developers.openai.com/docs/guides/conversation-state).

id: string

The unique identifier of the reasoning content.

summary: array of [SummaryTextContent](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20summary_text_content%20%3E%20\(schema\)) { text, type }

Reasoning summary content.

text: string

A summary of the reasoning output from the model so far.

type: "summary\_text"

The type of the object. Always `summary_text`.

type: "reasoning"

The type of the object. Always `reasoning`.

content: optional array of object { text, type }

Reasoning text content.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

encrypted\_content: optional string

The encrypted content of the reasoning item - populated when a response is generated with `reasoning.encrypted_content` in the `include` parameter.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

ToolSearchCall object { id, arguments, call\_id, 4 more }

id: string

The unique ID of the tool search call item.

arguments: unknown

Arguments used for the tool search call.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search call item that was recorded.

type: "tool\_search\_call"

The type of the item. Always `tool_search_call`.

created\_by: optional string

The identifier of the actor that created the item.

ToolSearchOutput object { id, call\_id, execution, 4 more }

id: string

The unique ID of the tool search output item.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search output item that was recorded.

tools: array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

The loaded tool definitions returned by tool search.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

type: "tool\_search\_output"

The type of the item. Always `tool_search_output`.

created\_by: optional string

The identifier of the actor that created the item.

Compaction object { id, encrypted\_content, type, created\_by }

id: string

The unique ID of the compaction item.

encrypted\_content: string

The encrypted content that was produced by compaction.

type: "compaction"

The type of the item. Always `compaction`.

created\_by: optional string

The identifier of the actor that created the item.

ImageGenerationCall object { id, result, status, type }

An image generation request made by the model.

id: string

The unique ID of the image generation call.

result: string

The generated image encoded in base64.

status: "in\_progress" or "completed" or "generating" or "failed"

The status of the image generation call.

type: "image\_generation\_call"

The type of the image generation call. Always `image_generation_call`.

CodeInterpreterCall object { id, code, container\_id, 3 more }

A tool call to run code.

id: string

The unique ID of the code interpreter tool call.

code: string

The code to run, or null if not available.

container\_id: string

The ID of the container used to run the code.

outputs: array of object { logs, type } or object { type, url }

The outputs generated by the code interpreter, such as logs or images. Can be null if no outputs are available.

One of the following:

Logs object { logs, type }

The logs output from the code interpreter.

logs: string

The logs output from the code interpreter.

type: "logs"

The type of the output. Always `logs`.

Image object { type, url }

The image output from the code interpreter.

type: "image"

The type of the output. Always `image`.

url: string

The URL of the image output from the code interpreter.

formaturi

status: "in\_progress" or "completed" or "incomplete" or 2 more

The status of the code interpreter tool call. Valid values are `in_progress`, `completed`, `incomplete`, `interpreting`, and `failed`.

type: "code\_interpreter\_call"

The type of the code interpreter tool call. Always `code_interpreter_call`.

LocalShellCall object { id, action, call\_id, 2 more }

A tool call to run a command on the local shell.

id: string

The unique ID of the local shell call.

action: object { command, env, type, 3 more }

Execute a shell command on the server.

command: array of string

The command to run.

env: map\[string\]

Environment variables to set for the command.

type: "exec"

The type of the local shell action. Always `exec`.

timeout\_ms: optional number

Optional timeout in milliseconds for the command.

user: optional string

Optional user to run the command as.

working\_directory: optional string

Optional working directory to run the command in.

call\_id: string

The unique ID of the local shell tool call generated by the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the local shell call.

type: "local\_shell\_call"

The type of the local shell call. Always `local_shell_call`.

LocalShellCallOutput object { id, output, type, status }

The output of a local shell tool call.

id: string

The unique ID of the local shell tool call generated by the model.

output: string

A JSON string of the output of the local shell tool call.

type: "local\_shell\_call\_output"

The type of the local shell tool call output. Always `local_shell_call_output`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`.

ShellCall object { id, action, call\_id, 4 more }

A tool call that executes one or more shell commands in a managed environment.

id: string

The unique ID of the shell tool call. Populated when this item is returned via API.

action: object { commands, max\_output\_length, timeout\_ms }

The shell commands and limits that describe how to run the tool call.

commands: array of string

max\_output\_length: number

Optional maximum number of characters to return from each command.

timeout\_ms: number

Optional timeout in milliseconds for the commands.

call\_id: string

The unique ID of the shell tool call generated by the model.

environment: [ResponseLocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_local_environment%20%3E%20\(schema\)) { type } or [ResponseContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_container_reference%20%3E%20\(schema\)) { container\_id, type }

Represents the use of a local environment to perform shell actions.

One of the following:

ResponseLocalEnvironment object { type }

Represents the use of a local environment to perform shell actions.

type: "local"

The environment type. Always `local`.

ResponseContainerReference object { container\_id, type }

Represents a container created with /v1/containers.

container\_id: string

type: "container\_reference"

The environment type. Always `container_reference`.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call"

The type of the item. Always `shell_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ShellCallOutput object { id, call\_id, max\_output\_length, 4 more }

The output of a shell tool call that was emitted.

id: string

The unique ID of the shell call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the shell tool call generated by the model.

max\_output\_length: number

The maximum length of the shell command output. This is generated by the model and should be passed back with the raw output.

output: array of object { outcome, stderr, stdout, created\_by }

An array of shell call output contents

outcome: object { type } or object { exit\_code, type }

Represents either an exit outcome (with an exit code) or a timeout outcome for a shell call output chunk.

One of the following:

Timeout object { type }

Indicates that the shell call exceeded its configured time limit.

type: "timeout"

The outcome type. Always `timeout`.

Exit object { exit\_code, type }

Indicates that the shell commands finished and returned an exit code.

exit\_code: number

Exit code from the shell process.

type: "exit"

The outcome type. Always `exit`.

stderr: string

The standard error output that was captured.

stdout: string

The standard output that was captured.

created\_by: optional string

The identifier of the actor that created the item.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call output. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call\_output"

The type of the shell call output. Always `shell_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

ApplyPatchCall object { id, call\_id, operation, 3 more }

A tool call that applies file diffs by creating, deleting, or updating files.

id: string

The unique ID of the apply patch tool call. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

operation: object { diff, path, type } or object { path, type } or object { diff, path, type }

One of the create\_file, delete\_file, or update\_file operations applied via apply\_patch.

One of the following:

CreateFile object { diff, path, type }

Instruction describing how to create a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to create.

type: "create\_file"

Create a new file with the provided diff.

DeleteFile object { path, type }

Instruction describing how to delete a file via the apply\_patch tool.

path: string

Path of the file to delete.

type: "delete\_file"

Delete the specified file.

UpdateFile object { diff, path, type }

Instruction describing how to update a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to update.

type: "update\_file"

Update an existing file with the provided diff.

status: "in\_progress" or "completed"

The status of the apply patch tool call. One of `in_progress` or `completed`.

One of the following:

"in\_progress"

"completed"

type: "apply\_patch\_call"

The type of the item. Always `apply_patch_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ApplyPatchCallOutput object { id, call\_id, status, 3 more }

The output emitted by an apply patch tool call.

id: string

The unique ID of the apply patch tool call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

status: "completed" or "failed"

The status of the apply patch tool call output. One of `completed` or `failed`.

One of the following:

"completed"

"failed"

type: "apply\_patch\_call\_output"

The type of the item. Always `apply_patch_call_output`.

created\_by: optional string

The ID of the entity that created this tool call output.

output: optional string

Optional textual output returned by the apply patch tool.

McpCall object { id, arguments, name, 6 more }

An invocation of a tool on an MCP server.

id: string

The unique ID of the tool call.

arguments: string

A JSON string of the arguments passed to the tool.

name: string

The name of the tool that was run.

server\_label: string

The label of the MCP server running the tool.

type: "mcp\_call"

The type of the item. Always `mcp_call`.

approval\_request\_id: optional string

Unique identifier for the MCP tool call approval request. Include this value in a subsequent `mcp_approval_response` input to approve or reject the corresponding tool call.

error: optional string

The error from the tool call, if any.

output: optional string

The output from the tool call.

status: optional "in\_progress" or "completed" or "incomplete" or 2 more

The status of the tool call. One of `in_progress`, `completed`, `incomplete`, `calling`, or `failed`.

McpListTools object { id, server\_label, tools, 2 more }

A list of tools available on an MCP server.

id: string

The unique ID of the list.

server\_label: string

The label of the MCP server.

tools: array of object { input\_schema, name, annotations, description }

The tools available on the server.

input\_schema: unknown

The JSON schema describing the tool’s input.

name: string

The name of the tool.

annotations: optional unknown

Additional annotations about the tool.

description: optional string

The description of the tool.

type: "mcp\_list\_tools"

The type of the item. Always `mcp_list_tools`.

error: optional string

Error message if the server could not list tools.

McpApprovalRequest object { id, arguments, name, 2 more }

A request for human approval of a tool invocation.

id: string

The unique ID of the approval request.

arguments: string

A JSON string of arguments for the tool.

name: string

The name of the tool to run.

server\_label: string

The label of the MCP server making the request.

type: "mcp\_approval\_request"

The type of the item. Always `mcp_approval_request`.

McpApprovalResponse object { id, approval\_request\_id, approve, 2 more }

A response to an MCP approval request.

id: string

The unique ID of the approval response

approval\_request\_id: string

The ID of the approval request being answered.

approve: boolean

Whether the request was approved.

type: "mcp\_approval\_response"

The type of the item. Always `mcp_approval_response`.

reason: optional string

Optional reason for the decision.

CustomToolCall object { call\_id, input, name, 3 more }

A call to a custom tool created by the model.

call\_id: string

An identifier used to map this custom tool call to a tool call output.

input: string

The input for the custom tool call generated by the model.

name: string

The name of the custom tool being called.

type: "custom\_tool\_call"

The type of the custom tool call. Always `custom_tool_call`.

id: optional string

The unique ID of the custom tool call in the OpenAI platform.

namespace: optional string

The namespace of the custom tool being called.

CustomToolCallOutput object { id, call\_id, output, 3 more }

id: string

The unique ID of the custom tool call output item.

call\_id: string

The call ID, used to map this custom tool call output to a custom tool call.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the custom tool call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the custom tool call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the custom tool call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "custom\_tool\_call\_output"

The type of the custom tool call output. Always `custom_tool_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

parallel\_tool\_calls: boolean

Whether to allow the model to run tool calls in parallel.

temperature: number

What sampling temperature to use, between 0 and 2. Higher values like 0.8 will make the output more random, while lower values like 0.2 will make it more focused and deterministic. We generally recommend altering this or `top_p` but not both.

minimum0

maximum2

tool\_choice: [ToolChoiceOptions](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20tool_choice_options%20%3E%20\(schema\)) or [ToolChoiceAllowed](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20tool_choice_allowed%20%3E%20\(schema\)) { mode, tools, type } or [ToolChoiceTypes](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20tool_choice_types%20%3E%20\(schema\)) { type } or 5 more

How the model should select which tool (or tools) to use when generating a response. See the `tools` parameter to see how to specify which tools the model can call.

One of the following:

ToolChoiceOptions = "none" or "auto" or "required"

Controls which (if any) tool is called by the model.

`none` means the model will not call any tool and instead generates a message.

`auto` means the model can pick between generating a message or calling one or more tools.

`required` means the model must call one or more tools.

ToolChoiceAllowed object { mode, tools, type }

Constrains the tools available to the model to a pre-defined set.

mode: "auto" or "required"

Constrains the tools available to the model to a pre-defined set.

`auto` allows the model to pick from among the allowed tools and generate a message.

`required` requires the model to call one or more of the allowed tools.

One of the following:

"auto"

"required"

A list of tool definitions that the model should be allowed to call.

For the Responses API, the list of tool definitions might look like:

```json
[
  { "type": "function", "name": "get_weather" },
  { "type": "mcp", "server_label": "deepwiki" },
  { "type": "image_generation" }
]
```

type: "allowed\_tools"

Allowed tool configuration type. Always `allowed_tools`.

ToolChoiceTypes object { type }

Indicates that the model should use a built-in tool to generate a response. [Learn more about built-in tools](https://developers.openai.com/docs/guides/tools).

type: "file\_search" or "web\_search\_preview" or "computer" or 5 more

The type of hosted tool the model should to use. Learn more about [built-in tools](https://developers.openai.com/docs/guides/tools).

Allowed values are:

- `file_search`
- `web_search_preview`
- `computer`
- `computer_use_preview`
- `computer_use`
- `code_interpreter`
- `image_generation`

ToolChoiceFunction object { name, type }

Use this option to force the model to call a specific function.

name: string

The name of the function to call.

type: "function"

For function calling, the type is always `function`.

ToolChoiceMcp object { server\_label, type, name }

Use this option to force the model to call a specific tool on a remote MCP server.

server\_label: string

The label of the MCP server to use.

type: "mcp"

For MCP tools, the type is always `mcp`.

name: optional string

The name of the tool to call on the server.

ToolChoiceCustom object { name, type }

Use this option to force the model to call a specific custom tool.

name: string

The name of the custom tool to call.

type: "custom"

For custom tool calling, the type is always `custom`.

ToolChoiceApplyPatch object { type }

Forces the model to call the apply\_patch tool when executing a tool call.

type: "apply\_patch"

The tool to call. Always `apply_patch`.

ToolChoiceShell object { type }

Forces the model to call the shell tool when a tool call is required.

type: "shell"

The tool to call. Always `shell`.

tools: array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

An array of tools the model may call while generating a response. You can specify which tool to use by setting the `tool_choice` parameter.

We support the following categories of tools:

- **Built-in tools**: Tools that are provided by OpenAI that extend the model’s capabilities, like [web search](https://developers.openai.com/docs/guides/tools-web-search) or [file search](https://developers.openai.com/docs/guides/tools-file-search). Learn more about [built-in tools](https://developers.openai.com/docs/guides/tools).
- **MCP Tools**: Integrations with third-party systems via custom MCP servers or predefined connectors such as Google Drive and SharePoint. Learn more about [MCP Tools](https://developers.openai.com/docs/guides/tools-connectors-mcp).
- **Function calls (custom tools)**: Functions that are defined by you, enabling the model to call your own code with strongly typed arguments and outputs. Learn more about [function calling](https://developers.openai.com/docs/guides/function-calling). You can also use custom tools to call your own code.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

top\_p: number

An alternative to sampling with temperature, called nucleus sampling, where the model considers the results of the tokens with top\_p probability mass. So 0.1 means only the tokens comprising the top 10% probability mass are considered.

We generally recommend altering this or `temperature` but not both.

minimum0

maximum1

background: optional boolean

Whether to run the model response in the background. [Learn more](https://developers.openai.com/docs/guides/background).

completed\_at: optional number

Unix timestamp (in seconds) of when this Response was completed. Only present when the status is `completed`.

formatunixtime

conversation: optional object { id }

The conversation that this response belonged to. Input items and output items from this response were automatically added to this conversation.

id: string

The unique ID of the conversation that this response was associated with.

max\_output\_tokens: optional number

An upper bound for the number of tokens that can be generated for a response, including visible output tokens and [reasoning tokens](https://developers.openai.com/docs/guides/reasoning).

max\_tool\_calls: optional number

The maximum number of total calls to built-in tools that can be processed in a response. This maximum number applies across all built-in tool calls, not per individual tool. Any further attempts to call a tool by the model will be ignored.

output\_text: optional string

SDK-only convenience property that contains the aggregated text output from all `output_text` items in the `output` array, if any are present. Supported in the Python and JavaScript SDKs.

previous\_response\_id: optional string

The unique ID of the previous response to the model. Use this to create multi-turn conversations. Learn more about [conversation state](https://developers.openai.com/docs/guides/conversation-state). Cannot be used in conjunction with `conversation`.

prompt: optional [ResponsePrompt](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_prompt%20%3E%20\(schema\)) { id, variables, version }

Reference to a prompt template and its variables. [Learn more](https://developers.openai.com/docs/guides/text?api-mode=responses#reusable-prompts).

prompt\_cache\_key: optional string

Used by OpenAI to cache responses for similar requests to optimize your cache hit rates. Replaces the `user` field. [Learn more](https://developers.openai.com/docs/guides/prompt-caching).

prompt\_cache\_retention: optional "in\_memory" or "24h"

The retention policy for the prompt cache. Set to `24h` to enable extended prompt caching, which keeps cached prefixes active for longer, up to a maximum of 24 hours. [Learn more](https://developers.openai.com/docs/guides/prompt-caching#prompt-cache-retention).

One of the following:

"in\_memory"

"24h"

reasoning: optional [Reasoning](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20reasoning%20%3E%20\(schema\)) { effort, generate\_summary, summary }

**gpt-5 and o-series models only**

Configuration options for [reasoning models](https://platform.openai.com/docs/guides/reasoning).

safety\_identifier: optional string

A stable identifier used to help detect users of your application that may be violating OpenAI’s usage policies. The IDs should be a string that uniquely identifies each user, with a maximum length of 64 characters. We recommend hashing their username or email address, in order to avoid sending us any identifying information. [Learn more](https://developers.openai.com/docs/guides/safety-best-practices#safety-identifiers).

maxLength64

service\_tier: optional "auto" or "default" or "flex" or 2 more

Specifies the processing type used for serving the request.

- If set to ‘auto’, then the request will be processed with the service tier configured in the Project settings. Unless otherwise configured, the Project will use ‘default’.
- If set to ‘default’, then the request will be processed with the standard pricing and performance for the selected model.
- If set to ‘ [flex](https://developers.openai.com/docs/guides/flex-processing) ’ or ‘ [priority](https://openai.com/api-priority-processing/) ’, then the request will be processed with the corresponding service tier.
- When not set, the default behavior is ‘auto’.

When the `service_tier` parameter is set, the response body will include the `service_tier` value based on the processing mode actually used to serve the request. This response value may be different from the value set in the parameter.

status: optional [ResponseStatus](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_status%20%3E%20\(schema\))

The status of the response generation. One of `completed`, `failed`, `in_progress`, `cancelled`, `queued`, or `incomplete`.

text: optional [ResponseTextConfig](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_text_config%20%3E%20\(schema\)) { format, verbosity }

Configuration options for a text response from the model. Can be plain text or structured JSON data. Learn more:

- [Text inputs and outputs](https://developers.openai.com/docs/guides/text)
- [Structured Outputs](https://developers.openai.com/docs/guides/structured-outputs)

top\_logprobs: optional number

An integer between 0 and 20 specifying the maximum number of most likely tokens to return at each token position, each with an associated log probability. In some cases, the number of returned tokens may be fewer than requested.

minimum0

maximum20

truncation: optional "auto" or "disabled"

The truncation strategy to use for the model response.

- `auto`: If the input to this Response exceeds the model’s context window size, the model will truncate the response to fit the context window by dropping items from the beginning of the conversation.
- `disabled` (default): If the input size will exceed the context window size for a model, the request will fail with a 400 error.

One of the following:

"auto"

"disabled"

usage: optional [ResponseUsage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_usage%20%3E%20\(schema\)) { input\_tokens, input\_tokens\_details, output\_tokens, 2 more }

Represents token usage details including input tokens, output tokens, a breakdown of output tokens, and the total tokens used.

Deprecateduser: optional string

This field is being replaced by `safety_identifier` and `prompt_cache_key`. Use `prompt_cache_key` instead to maintain caching optimizations. A stable identifier for your end-users. Used to boost cache hit rates by better bucketing similar requests and to help OpenAI detect and prevent abuse. [Learn more](https://developers.openai.com/docs/guides/safety-best-practices#safety-identifiers).

ResponseAudioDeltaEvent object { delta, sequence\_number, type }

Emitted when there is a partial audio response.

delta: string

A chunk of Base64 encoded response audio bytes.

sequence\_number: number

A sequence number for this chunk of the stream response.

type: "response.audio.delta"

The type of the event. Always `response.audio.delta`.

ResponseAudioDoneEvent object { sequence\_number, type }

Emitted when the audio response is complete.

sequence\_number: number

The sequence number of the delta.

type: "response.audio.done"

The type of the event. Always `response.audio.done`.

ResponseAudioTranscriptDeltaEvent object { delta, sequence\_number, type }

Emitted when there is a partial transcript of audio.

delta: string

The partial transcript of the audio response.

sequence\_number: number

The sequence number of this event.

type: "response.audio.transcript.delta"

The type of the event. Always `response.audio.transcript.delta`.

ResponseAudioTranscriptDoneEvent object { sequence\_number, type }

Emitted when the full audio transcript is completed.

sequence\_number: number

The sequence number of this event.

type: "response.audio.transcript.done"

The type of the event. Always `response.audio.transcript.done`.

ResponseCodeInterpreterCallCodeDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when a partial code snippet is streamed by the code interpreter.

delta: string

The partial code snippet being streamed by the code interpreter.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code is being streamed.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call\_code.delta"

The type of the event. Always `response.code_interpreter_call_code.delta`.

ResponseCodeInterpreterCallCodeDoneEvent object { code, item\_id, output\_index, 2 more }

Emitted when the code snippet is finalized by the code interpreter.

code: string

The final code snippet output by the code interpreter.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code is finalized.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call\_code.done"

The type of the event. Always `response.code_interpreter_call_code.done`.

ResponseCodeInterpreterCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the code interpreter call is completed.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter call is completed.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.completed"

The type of the event. Always `response.code_interpreter_call.completed`.

ResponseCodeInterpreterCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when a code interpreter call is in progress.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter call is in progress.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.in\_progress"

The type of the event. Always `response.code_interpreter_call.in_progress`.

ResponseCodeInterpreterCallInterpretingEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the code interpreter is actively interpreting the code snippet.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter is interpreting code.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.interpreting"

The type of the event. Always `response.code_interpreter_call.interpreting`.

ResponseCompletedEvent object { response, sequence\_number, type }

Emitted when the model response is complete.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

Properties of the completed response.

sequence\_number: number

The sequence number for this event.

type: "response.completed"

The type of the event. Always `response.completed`.

ResponseComputerToolCallOutputScreenshot object { type, file\_id, image\_url }

A computer screenshot image used with the computer use tool.

type: "computer\_screenshot"

Specifies the event type. For a computer screenshot, this property is always set to `computer_screenshot`.

file\_id: optional string

The identifier of an uploaded file that contains the screenshot.

image\_url: optional string

The URL of the screenshot image.

formaturi

ResponseContainerReference object { container\_id, type }

Represents a container created with /v1/containers.

container\_id: string

type: "container\_reference"

The environment type. Always `container_reference`.

ResponseContent = [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more } or 3 more

Multi-modal input and output contents.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

ResponseContentPartAddedEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a new content part is added.

content\_index: number

The index of the content part that was added.

item\_id: string

The ID of the output item that the content part was added to.

output\_index: number

The index of the output item that the content part was added to.

part: [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type } or object { text, type }

The content part that was added.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

sequence\_number: number

The sequence number of this event.

type: "response.content\_part.added"

The type of the event. Always `response.content_part.added`.

ResponseContentPartDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a content part is done.

content\_index: number

The index of the content part that is done.

item\_id: string

The ID of the output item that the content part was added to.

output\_index: number

The index of the output item that the content part was added to.

part: [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type } or object { text, type }

The content part that is done.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

sequence\_number: number

The sequence number of this event.

type: "response.content\_part.done"

The type of the event. Always `response.content_part.done`.

ResponseConversationParam object { id }

The conversation that this response belongs to.

id: string

The unique ID of the conversation.

ResponseCreatedEvent object { response, sequence\_number, type }

An event that is emitted when a response is created.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that was created.

sequence\_number: number

The sequence number for this event.

type: "response.created"

The type of the event. Always `response.created`.

ResponseCustomToolCallInputDeltaEvent object { delta, item\_id, output\_index, 2 more }

Event representing a delta (partial update) to the input of a custom tool call.

delta: string

The incremental input data (delta) for the custom tool call.

item\_id: string

Unique identifier for the API item associated with this event.

output\_index: number

The index of the output this delta applies to.

sequence\_number: number

The sequence number of this event.

type: "response.custom\_tool\_call\_input.delta"

The event type identifier.

ResponseCustomToolCallInputDoneEvent object { input, item\_id, output\_index, 2 more }

Event indicating that input for a custom tool call is complete.

input: string

The complete input data for the custom tool call.

item\_id: string

Unique identifier for the API item associated with this event.

output\_index: number

The index of the output this event applies to.

sequence\_number: number

The sequence number of this event.

type: "response.custom\_tool\_call\_input.done"

The event type identifier.

ResponseError object { code, message }

An error object returned when the model fails to generate a Response.

ResponseErrorEvent object { code, message, param, 2 more }

Emitted when an error occurs.

code: string

The error code.

message: string

The error message.

param: string

The error parameter.

sequence\_number: number

The sequence number of this event.

type: "error"

The type of the event. Always `error`.

ResponseFailedEvent object { response, sequence\_number, type }

An event that is emitted when a response fails.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that failed.

sequence\_number: number

The sequence number of this event.

type: "response.failed"

The type of the event. Always `response.failed`.

ResponseFormatTextConfig = [ResponseFormatText](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20response_format_text%20%3E%20\(schema\)) { type } or [ResponseFormatTextJSONSchemaConfig](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_format_text_json_schema_config%20%3E%20\(schema\)) { name, schema, type, 2 more } or [ResponseFormatJSONObject](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20response_format_json_object%20%3E%20\(schema\)) { type }

An object specifying the format that the model must output.

Configuring `{ "type": "json_schema" }` enables Structured Outputs, which ensures the model will match your supplied JSON schema. Learn more in the [Structured Outputs guide](https://developers.openai.com/docs/guides/structured-outputs).

The default format is `{ "type": "text" }` with no additional options.

**Not recommended for gpt-4o and newer models:**

Setting to `{ "type": "json_object" }` enables the older JSON mode, which ensures the message the model generates is valid JSON. Using `json_schema` is preferred for models that support it.

One of the following:

ResponseFormatText object { type }

type: "text"

The type of response format being defined. Always `text`.

ResponseFormatTextJSONSchemaConfig object { name, schema, type, 2 more }

JSON Schema response format. Used to generate structured JSON responses. Learn more about [Structured Outputs](https://developers.openai.com/docs/guides/structured-outputs).

name: string

The name of the response format. Must be a-z, A-Z, 0-9, or contain underscores and dashes, with a maximum length of 64.

schema: map\[unknown\]

The schema for the response format, described as a JSON Schema object. Learn how to build JSON schemas [here](https://json-schema.org/).

type: "json\_schema"

The type of response format being defined. Always `json_schema`.

description: optional string

A description of what the response format is for, used by the model to determine how to respond in the format.

strict: optional boolean

Whether to enable strict schema adherence when generating the output. If set to true, the model will always follow the exact schema defined in the `schema` field. Only a subset of JSON Schema is supported when `strict` is `true`. To learn more, read the [Structured Outputs guide](https://developers.openai.com/docs/guides/structured-outputs).

ResponseFormatJSONObject object { type }

JSON object response format. An older method of generating JSON responses. Using `json_schema` is recommended for models that support it. Note that the model will not generate JSON without a system or user message instructing it to do so.

type: "json\_object"

The type of response format being defined. Always `json_object`.

ResponseFormatTextJSONSchemaConfig object { name, schema, type, 2 more }

JSON Schema response format. Used to generate structured JSON responses. Learn more about [Structured Outputs](https://developers.openai.com/docs/guides/structured-outputs).

name: string

The name of the response format. Must be a-z, A-Z, 0-9, or contain underscores and dashes, with a maximum length of 64.

schema: map\[unknown\]

The schema for the response format, described as a JSON Schema object. Learn how to build JSON schemas [here](https://json-schema.org/).

type: "json\_schema"

The type of response format being defined. Always `json_schema`.

description: optional string

A description of what the response format is for, used by the model to determine how to respond in the format.

strict: optional boolean

Whether to enable strict schema adherence when generating the output. If set to true, the model will always follow the exact schema defined in the `schema` field. Only a subset of JSON Schema is supported when `strict` is `true`. To learn more, read the [Structured Outputs guide](https://developers.openai.com/docs/guides/structured-outputs).

ResponseFunctionCallArgumentsDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when there is a partial function-call arguments delta.

delta: string

The function-call arguments delta that is added.

item\_id: string

The ID of the output item that the function-call arguments delta is added to.

output\_index: number

The index of the output item that the function-call arguments delta is added to.

sequence\_number: number

The sequence number of this event.

type: "response.function\_call\_arguments.delta"

The type of the event. Always `response.function_call_arguments.delta`.

ResponseFunctionCallArgumentsDoneEvent object { arguments, item\_id, name, 3 more }

Emitted when function-call arguments are finalized.

arguments: string

The function-call arguments.

item\_id: string

The ID of the item.

name: string

The name of the function that was called.

output\_index: number

The index of the output item.

sequence\_number: number

The sequence number of this event.

type: "response.function\_call\_arguments.done"

ResponseFunctionShellCallOutputContent object { outcome, stderr, stdout }

Captured stdout and stderr for a portion of a shell tool call output.

outcome: object { type } or object { exit\_code, type }

The exit or timeout outcome associated with this shell call.

One of the following:

Timeout object { type }

Indicates that the shell call exceeded its configured time limit.

type: "timeout"

The outcome type. Always `timeout`.

Exit object { exit\_code, type }

Indicates that the shell commands finished and returned an exit code.

exit\_code: number

The exit code returned by the shell process.

type: "exit"

The outcome type. Always `exit`.

stderr: string

Captured stderr output for the shell call.

maxLength10485760

stdout: string

Captured stdout output for the shell call.

maxLength10485760

ResponseImageGenCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call has completed and the final image is available.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.image\_generation\_call.completed"

The type of the event. Always ‘response.image\_generation\_call.completed’.

ResponseImageGenCallGeneratingEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call is actively generating an image (intermediate state).

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.generating"

The type of the event. Always ‘response.image\_generation\_call.generating’.

ResponseImageGenCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call is in progress.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.in\_progress"

The type of the event. Always ‘response.image\_generation\_call.in\_progress’.

ResponseImageGenCallPartialImageEvent object { item\_id, output\_index, partial\_image\_b64, 3 more }

Emitted when a partial image is available during image generation streaming.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

partial\_image\_b64: string

Base64-encoded partial image data, suitable for rendering as an image.

partial\_image\_index: number

0-based index for the partial image (backend is 1-based, but this is 0-based for the user).

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.partial\_image"

The type of the event. Always ‘response.image\_generation\_call.partial\_image’.

ResponseInProgressEvent object { response, sequence\_number, type }

Emitted when the response is in progress.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that is in progress.

sequence\_number: number

The sequence number of this event.

type: "response.in\_progress"

The type of the event. Always `response.in_progress`.

ResponseIncludable = "file\_search\_call.results" or "web\_search\_call.results" or "web\_search\_call.action.sources" or 5 more

Specify additional output data to include in the model response. Currently supported values are:

- `web_search_call.results`: Include the search results of the web search tool call.
- `web_search_call.action.sources`: Include the sources of the web search tool call.
- `code_interpreter_call.outputs`: Includes the outputs of python code execution in code interpreter tool call items.
- `computer_call_output.output.image_url`: Include image urls from the computer call output.
- `file_search_call.results`: Include the search results of the file search tool call.
- `message.input_image.image_url`: Include image urls from the input message.
- `message.output_text.logprobs`: Include logprobs with assistant messages.
- `reasoning.encrypted_content`: Includes an encrypted version of reasoning tokens in reasoning item outputs. This enables reasoning items to be used in multi-turn conversations when using the Responses API statelessly (like when the `store` parameter is set to `false`, or when an organization is enrolled in the zero data retention program).

ResponseIncompleteEvent object { response, sequence\_number, type }

An event that is emitted when a response finishes as incomplete.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that was incomplete.

sequence\_number: number

The sequence number of this event.

type: "response.incomplete"

The type of the event. Always `response.incomplete`.

ResponseInputAudio object { input\_audio, type }

An audio input to the model.

input\_audio: object { data, format }

data: string

Base64-encoded audio data.

format: "mp3" or "wav"

The format of the audio data. Currently supported formats are `mp3` and `wav`.

One of the following:

"mp3"

"wav"

type: "input\_audio"

The type of the input item. Always `input_audio`.

ResponseInputContent = [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

A text input to the model.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

ResponseInputFileContent object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The base64-encoded data of the file to be sent to the model.

maxLength73400320

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputImageContent object { type, detail, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision)

type: "input\_image"

The type of the input item. Always `input_image`.

detail: optional "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

maxLength20971520

formaturi

ResponseInputMessageContentList = array of [ResponseInputContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_content%20%3E%20\(schema\))

A list of one or many input items to the model, containing different content types.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

ResponseInputMessageItem object { id, content, role, 2 more }

id: string

The unique ID of the message input.

content: [ResponseInputMessageContentList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_content_list%20%3E%20\(schema\)) {,, }

A list of one or many input items to the model, containing different content types.

role: "user" or "system" or "developer"

The role of the message input. One of `user`, `system`, or `developer`.

type: "message"

The type of the message input. Always set to `message`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputTextContent object { text, type }

A text input to the model.

text: string

The text input to the model.

maxLength10485760

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseLocalEnvironment object { type }

Represents the use of a local environment to perform shell actions.

type: "local"

The environment type. Always `local`.

ResponseMcpCallArgumentsDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when there is a delta (partial update) to the arguments of an MCP tool call.

delta: string

A JSON string containing the partial update to the arguments for the MCP tool call.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call\_arguments.delta"

The type of the event. Always ‘response.mcp\_call\_arguments.delta’.

ResponseMcpCallArgumentsDoneEvent object { arguments, item\_id, output\_index, 2 more }

Emitted when the arguments for an MCP tool call are finalized.

arguments: string

A JSON string containing the finalized arguments for the MCP tool call.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call\_arguments.done"

The type of the event. Always ‘response.mcp\_call\_arguments.done’.

ResponseMcpCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call has completed successfully.

item\_id: string

The ID of the MCP tool call item that completed.

output\_index: number

The index of the output item that completed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.completed"

The type of the event. Always ‘response.mcp\_call.completed’.

ResponseMcpCallFailedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call has failed.

item\_id: string

The ID of the MCP tool call item that failed.

output\_index: number

The index of the output item that failed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.failed"

The type of the event. Always ‘response.mcp\_call.failed’.

ResponseMcpCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call is in progress.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.in\_progress"

The type of the event. Always ‘response.mcp\_call.in\_progress’.

ResponseMcpListToolsCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the list of available MCP tools has been successfully retrieved.

item\_id: string

The ID of the MCP tool call item that produced this output.

output\_index: number

The index of the output item that was processed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.completed"

The type of the event. Always ‘response.mcp\_list\_tools.completed’.

ResponseMcpListToolsFailedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the attempt to list available MCP tools has failed.

item\_id: string

The ID of the MCP tool call item that failed.

output\_index: number

The index of the output item that failed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.failed"

The type of the event. Always ‘response.mcp\_list\_tools.failed’.

ResponseMcpListToolsInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the system is in the process of retrieving the list of available MCP tools.

item\_id: string

The ID of the MCP tool call item that is being processed.

output\_index: number

The index of the output item that is being processed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.in\_progress"

The type of the event. Always ‘response.mcp\_list\_tools.in\_progress’.

ResponseOutputAudio object { data, transcript, type }

An audio output from the model.

data: string

Base64-encoded audio data from the model.

transcript: string

The transcript of the audio data from the model.

type: "output\_audio"

The type of the output audio. Always `output_audio`.

ResponseOutputItem = [ResponseOutputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_message%20%3E%20\(schema\)) { id, content, role, 3 more } or object { id, queries, status, 2 more } or object { arguments, call\_id, name, 4 more } or 22 more

An output message from the model.

One of the following:

ResponseOutputMessage object { id, content, role, 3 more }

An output message from the model.

id: string

The unique ID of the output message.

content: array of [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type }

The content of the output message.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

role: "assistant"

The role of the output message. Always `assistant`.

status: "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "message"

The type of the output message. Always `message`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

FileSearchCall object { id, queries, status, 2 more }

The results of a file search tool call. See the [file search guide](https://developers.openai.com/docs/guides/tools-file-search) for more information.

id: string

The unique ID of the file search tool call.

queries: array of string

The queries used to search for files.

status: "in\_progress" or "searching" or "completed" or 2 more

The status of the file search tool call. One of `in_progress`, `searching`, `incomplete` or `failed`,

type: "file\_search\_call"

The type of the file search tool call. Always `file_search_call`.

results: optional array of object { attributes, file\_id, filename, 2 more }

The results of the file search tool call.

attributes: optional map\[string or number or boolean\]

Set of 16 key-value pairs that can be attached to an object. This can be useful for storing additional information about the object in a structured format, and querying for objects via API or the dashboard. Keys are strings with a maximum length of 64 characters. Values are strings with a maximum length of 512 characters, booleans, or numbers.

file\_id: optional string

The unique ID of the file.

filename: optional string

The name of the file.

score: optional number

The relevance score of the file - a value between 0 and 1.

formatfloat

text: optional string

The text that was retrieved from the file.

FunctionCall object { arguments, call\_id, name, 4 more }

A tool call to run a function. See the [function calling guide](https://developers.openai.com/docs/guides/function-calling) for more information.

arguments: string

A JSON string of the arguments to pass to the function.

call\_id: string

The unique ID of the function tool call generated by the model.

name: string

The name of the function to run.

type: "function\_call"

The type of the function tool call. Always `function_call`.

id: optional string

The unique ID of the function tool call.

namespace: optional string

The namespace of the function to run.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

FunctionCallOutput object { id, call\_id, output, 3 more }

id: string

The unique ID of the function call tool output.

call\_id: string

The unique ID of the function tool call generated by the model.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the function call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the function call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the function call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "function\_call\_output"

The type of the function tool call output. Always `function_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

WebSearchCall object { id, action, status, type }

The results of a web search tool call. See the [web search guide](https://developers.openai.com/docs/guides/tools-web-search) for more information.

id: string

The unique ID of the web search tool call.

action: object { query, type, queries, sources } or object { type, url } or object { pattern, type, url }

An object describing the specific action taken in this web search call. Includes details on how the model used the web (search, open\_page, find\_in\_page).

One of the following:

Search object { query, type, queries, sources }

Action type “search” - Performs a web search query.

query: string

\[DEPRECATED\] The search query.

type: "search"

The action type.

queries: optional array of string

The search queries.

sources: optional array of object { type, url }

The sources used in the search.

type: "url"

The type of source. Always `url`.

url: string

The URL of the source.

formaturi

OpenPage object { type, url }

Action type “open\_page” - Opens a specific URL from search results.

type: "open\_page"

The action type.

url: optional string

The URL opened by the model.

formaturi

FindInPage object { pattern, type, url }

Action type “find\_in\_page”: Searches for a pattern within a loaded page.

pattern: string

The pattern or text to search for within the page.

type: "find\_in\_page"

The action type.

url: string

The URL of the page searched for the pattern.

formaturi

status: "in\_progress" or "searching" or "completed" or "failed"

The status of the web search tool call.

type: "web\_search\_call"

The type of the web search tool call. Always `web_search_call`.

ComputerCall object { id, call\_id, pending\_safety\_checks, 4 more }

A tool call to a computer use tool. See the [computer use guide](https://developers.openai.com/docs/guides/tools-computer-use) for more information.

id: string

The unique ID of the computer call.

call\_id: string

An identifier used when responding to the tool call with output.

pending\_safety\_checks: array of object { id, code, message }

The pending safety checks for the computer call.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "computer\_call"

The type of the computer call. Always `computer_call`.

action: optional [ComputerAction](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action%20%3E%20\(schema\))

A click action.

actions: optional [ComputerActionList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action_list%20%3E%20\(schema\)) { Click, DoubleClick, Drag, 6 more }

Flattened batched actions for `computer_use`. Each action includes an `type` discriminator and action-specific fields.

ComputerCallOutput object { id, call\_id, output, 4 more }

id: string

The unique ID of the computer call tool output.

call\_id: string

The ID of the computer tool call that produced the output.

output: [ResponseComputerToolCallOutputScreenshot](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_computer_tool_call_output_screenshot%20%3E%20\(schema\)) { type, file\_id, image\_url }

A computer screenshot image used with the computer use tool.

status: "completed" or "incomplete" or "failed" or "in\_progress"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "computer\_call\_output"

The type of the computer tool call output. Always `computer_call_output`.

acknowledged\_safety\_checks: optional array of object { id, code, message }

The safety checks reported by the API that have been acknowledged by the developer.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

created\_by: optional string

The identifier of the actor that created the item.

Reasoning object { id, summary, type, 3 more }

A description of the chain of thought used by a reasoning model while generating a response. Be sure to include these items in your `input` to the Responses API for subsequent turns of a conversation if you are manually [managing context](https://developers.openai.com/docs/guides/conversation-state).

id: string

The unique identifier of the reasoning content.

summary: array of [SummaryTextContent](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20summary_text_content%20%3E%20\(schema\)) { text, type }

Reasoning summary content.

text: string

A summary of the reasoning output from the model so far.

type: "summary\_text"

The type of the object. Always `summary_text`.

type: "reasoning"

The type of the object. Always `reasoning`.

content: optional array of object { text, type }

Reasoning text content.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

encrypted\_content: optional string

The encrypted content of the reasoning item - populated when a response is generated with `reasoning.encrypted_content` in the `include` parameter.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

ToolSearchCall object { id, arguments, call\_id, 4 more }

id: string

The unique ID of the tool search call item.

arguments: unknown

Arguments used for the tool search call.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search call item that was recorded.

type: "tool\_search\_call"

The type of the item. Always `tool_search_call`.

created\_by: optional string

The identifier of the actor that created the item.

ToolSearchOutput object { id, call\_id, execution, 4 more }

id: string

The unique ID of the tool search output item.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search output item that was recorded.

tools: array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

The loaded tool definitions returned by tool search.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

type: "tool\_search\_output"

The type of the item. Always `tool_search_output`.

created\_by: optional string

The identifier of the actor that created the item.

Compaction object { id, encrypted\_content, type, created\_by }

id: string

The unique ID of the compaction item.

encrypted\_content: string

The encrypted content that was produced by compaction.

type: "compaction"

The type of the item. Always `compaction`.

created\_by: optional string

The identifier of the actor that created the item.

ImageGenerationCall object { id, result, status, type }

An image generation request made by the model.

id: string

The unique ID of the image generation call.

result: string

The generated image encoded in base64.

status: "in\_progress" or "completed" or "generating" or "failed"

The status of the image generation call.

type: "image\_generation\_call"

The type of the image generation call. Always `image_generation_call`.

CodeInterpreterCall object { id, code, container\_id, 3 more }

A tool call to run code.

id: string

The unique ID of the code interpreter tool call.

code: string

The code to run, or null if not available.

container\_id: string

The ID of the container used to run the code.

outputs: array of object { logs, type } or object { type, url }

The outputs generated by the code interpreter, such as logs or images. Can be null if no outputs are available.

One of the following:

Logs object { logs, type }

The logs output from the code interpreter.

logs: string

The logs output from the code interpreter.

type: "logs"

The type of the output. Always `logs`.

Image object { type, url }

The image output from the code interpreter.

type: "image"

The type of the output. Always `image`.

url: string

The URL of the image output from the code interpreter.

formaturi

status: "in\_progress" or "completed" or "incomplete" or 2 more

The status of the code interpreter tool call. Valid values are `in_progress`, `completed`, `incomplete`, `interpreting`, and `failed`.

type: "code\_interpreter\_call"

The type of the code interpreter tool call. Always `code_interpreter_call`.

LocalShellCall object { id, action, call\_id, 2 more }

A tool call to run a command on the local shell.

id: string

The unique ID of the local shell call.

action: object { command, env, type, 3 more }

Execute a shell command on the server.

command: array of string

The command to run.

env: map\[string\]

Environment variables to set for the command.

type: "exec"

The type of the local shell action. Always `exec`.

timeout\_ms: optional number

Optional timeout in milliseconds for the command.

user: optional string

Optional user to run the command as.

working\_directory: optional string

Optional working directory to run the command in.

call\_id: string

The unique ID of the local shell tool call generated by the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the local shell call.

type: "local\_shell\_call"

The type of the local shell call. Always `local_shell_call`.

LocalShellCallOutput object { id, output, type, status }

The output of a local shell tool call.

id: string

The unique ID of the local shell tool call generated by the model.

output: string

A JSON string of the output of the local shell tool call.

type: "local\_shell\_call\_output"

The type of the local shell tool call output. Always `local_shell_call_output`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`.

ShellCall object { id, action, call\_id, 4 more }

A tool call that executes one or more shell commands in a managed environment.

id: string

The unique ID of the shell tool call. Populated when this item is returned via API.

action: object { commands, max\_output\_length, timeout\_ms }

The shell commands and limits that describe how to run the tool call.

commands: array of string

max\_output\_length: number

Optional maximum number of characters to return from each command.

timeout\_ms: number

Optional timeout in milliseconds for the commands.

call\_id: string

The unique ID of the shell tool call generated by the model.

environment: [ResponseLocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_local_environment%20%3E%20\(schema\)) { type } or [ResponseContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_container_reference%20%3E%20\(schema\)) { container\_id, type }

Represents the use of a local environment to perform shell actions.

One of the following:

ResponseLocalEnvironment object { type }

Represents the use of a local environment to perform shell actions.

type: "local"

The environment type. Always `local`.

ResponseContainerReference object { container\_id, type }

Represents a container created with /v1/containers.

container\_id: string

type: "container\_reference"

The environment type. Always `container_reference`.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call"

The type of the item. Always `shell_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ShellCallOutput object { id, call\_id, max\_output\_length, 4 more }

The output of a shell tool call that was emitted.

id: string

The unique ID of the shell call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the shell tool call generated by the model.

max\_output\_length: number

The maximum length of the shell command output. This is generated by the model and should be passed back with the raw output.

output: array of object { outcome, stderr, stdout, created\_by }

An array of shell call output contents

outcome: object { type } or object { exit\_code, type }

Represents either an exit outcome (with an exit code) or a timeout outcome for a shell call output chunk.

One of the following:

Timeout object { type }

Indicates that the shell call exceeded its configured time limit.

type: "timeout"

The outcome type. Always `timeout`.

Exit object { exit\_code, type }

Indicates that the shell commands finished and returned an exit code.

exit\_code: number

Exit code from the shell process.

type: "exit"

The outcome type. Always `exit`.

stderr: string

The standard error output that was captured.

stdout: string

The standard output that was captured.

created\_by: optional string

The identifier of the actor that created the item.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call output. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call\_output"

The type of the shell call output. Always `shell_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

ApplyPatchCall object { id, call\_id, operation, 3 more }

A tool call that applies file diffs by creating, deleting, or updating files.

id: string

The unique ID of the apply patch tool call. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

operation: object { diff, path, type } or object { path, type } or object { diff, path, type }

One of the create\_file, delete\_file, or update\_file operations applied via apply\_patch.

One of the following:

CreateFile object { diff, path, type }

Instruction describing how to create a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to create.

type: "create\_file"

Create a new file with the provided diff.

DeleteFile object { path, type }

Instruction describing how to delete a file via the apply\_patch tool.

path: string

Path of the file to delete.

type: "delete\_file"

Delete the specified file.

UpdateFile object { diff, path, type }

Instruction describing how to update a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to update.

type: "update\_file"

Update an existing file with the provided diff.

status: "in\_progress" or "completed"

The status of the apply patch tool call. One of `in_progress` or `completed`.

One of the following:

"in\_progress"

"completed"

type: "apply\_patch\_call"

The type of the item. Always `apply_patch_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ApplyPatchCallOutput object { id, call\_id, status, 3 more }

The output emitted by an apply patch tool call.

id: string

The unique ID of the apply patch tool call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

status: "completed" or "failed"

The status of the apply patch tool call output. One of `completed` or `failed`.

One of the following:

"completed"

"failed"

type: "apply\_patch\_call\_output"

The type of the item. Always `apply_patch_call_output`.

created\_by: optional string

The ID of the entity that created this tool call output.

output: optional string

Optional textual output returned by the apply patch tool.

McpCall object { id, arguments, name, 6 more }

An invocation of a tool on an MCP server.

id: string

The unique ID of the tool call.

arguments: string

A JSON string of the arguments passed to the tool.

name: string

The name of the tool that was run.

server\_label: string

The label of the MCP server running the tool.

type: "mcp\_call"

The type of the item. Always `mcp_call`.

approval\_request\_id: optional string

Unique identifier for the MCP tool call approval request. Include this value in a subsequent `mcp_approval_response` input to approve or reject the corresponding tool call.

error: optional string

The error from the tool call, if any.

output: optional string

The output from the tool call.

status: optional "in\_progress" or "completed" or "incomplete" or 2 more

The status of the tool call. One of `in_progress`, `completed`, `incomplete`, `calling`, or `failed`.

McpListTools object { id, server\_label, tools, 2 more }

A list of tools available on an MCP server.

id: string

The unique ID of the list.

server\_label: string

The label of the MCP server.

tools: array of object { input\_schema, name, annotations, description }

The tools available on the server.

input\_schema: unknown

The JSON schema describing the tool’s input.

name: string

The name of the tool.

annotations: optional unknown

Additional annotations about the tool.

description: optional string

The description of the tool.

type: "mcp\_list\_tools"

The type of the item. Always `mcp_list_tools`.

error: optional string

Error message if the server could not list tools.

McpApprovalRequest object { id, arguments, name, 2 more }

A request for human approval of a tool invocation.

id: string

The unique ID of the approval request.

arguments: string

A JSON string of arguments for the tool.

name: string

The name of the tool to run.

server\_label: string

The label of the MCP server making the request.

type: "mcp\_approval\_request"

The type of the item. Always `mcp_approval_request`.

McpApprovalResponse object { id, approval\_request\_id, approve, 2 more }

A response to an MCP approval request.

id: string

The unique ID of the approval response

approval\_request\_id: string

The ID of the approval request being answered.

approve: boolean

Whether the request was approved.

type: "mcp\_approval\_response"

The type of the item. Always `mcp_approval_response`.

reason: optional string

Optional reason for the decision.

CustomToolCall object { call\_id, input, name, 3 more }

A call to a custom tool created by the model.

call\_id: string

An identifier used to map this custom tool call to a tool call output.

input: string

The input for the custom tool call generated by the model.

name: string

The name of the custom tool being called.

type: "custom\_tool\_call"

The type of the custom tool call. Always `custom_tool_call`.

id: optional string

The unique ID of the custom tool call in the OpenAI platform.

namespace: optional string

The namespace of the custom tool being called.

CustomToolCallOutput object { id, call\_id, output, 3 more }

id: string

The unique ID of the custom tool call output item.

call\_id: string

The call ID, used to map this custom tool call output to a custom tool call.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the custom tool call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the custom tool call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the custom tool call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "custom\_tool\_call\_output"

The type of the custom tool call output. Always `custom_tool_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

ResponseOutputItemAddedEvent object { item, output\_index, sequence\_number, type }

Emitted when a new output item is added.

item: [ResponseOutputItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_item%20%3E%20\(schema\))

The output item that was added.

output\_index: number

The index of the output item that was added.

sequence\_number: number

The sequence number of this event.

type: "response.output\_item.added"

The type of the event. Always `response.output_item.added`.

ResponseOutputItemDoneEvent object { item, output\_index, sequence\_number, type }

Emitted when an output item is marked done.

item: [ResponseOutputItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_item%20%3E%20\(schema\))

The output item that was marked done.

output\_index: number

The index of the output item that was marked done.

sequence\_number: number

The sequence number of this event.

type: "response.output\_item.done"

The type of the event. Always `response.output_item.done`.

ResponseOutputMessage object { id, content, role, 3 more }

An output message from the model.

id: string

The unique ID of the output message.

content: array of [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type }

The content of the output message.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

role: "assistant"

The role of the output message. Always `assistant`.

status: "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "message"

The type of the output message. Always `message`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputTextAnnotationAddedEvent object { annotation, annotation\_index, content\_index, 4 more }

Emitted when an annotation is added to output text content.

annotation: unknown

The annotation object being added. (See annotation schema for details.)

annotation\_index: number

The index of the annotation within the content part.

content\_index: number

The index of the content part within the output item.

item\_id: string

The unique identifier of the item to which the annotation is being added.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.output\_text.annotation.added"

The type of the event. Always ‘response.output\_text.annotation.added’.

ResponsePrompt object { id, variables, version }

Reference to a prompt template and its variables. [Learn more](https://developers.openai.com/docs/guides/text?api-mode=responses#reusable-prompts).

id: string

The unique identifier of the prompt template to use.

variables: optional map\[string or [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more } \]

Optional map of values to substitute in for variables in your prompt. The substitution values can either be strings, or other Response input types like images or files.

One of the following:

string

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

version: optional string

Optional version of the prompt template.

ResponseQueuedEvent object { response, sequence\_number, type }

Emitted when a response is queued and waiting to be processed.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The full response object that is queued.

sequence\_number: number

The sequence number for this event.

type: "response.queued"

The type of the event. Always ‘response.queued’.

ResponseReasoningSummaryPartAddedEvent object { item\_id, output\_index, part, 3 more }

Emitted when a new reasoning summary part is added.

item\_id: string

The ID of the item this summary part is associated with.

output\_index: number

The index of the output item this summary part is associated with.

part: object { text, type }

The summary part that was added.

text: string

The text of the summary part.

type: "summary\_text"

The type of the summary part. Always `summary_text`.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_part.added"

The type of the event. Always `response.reasoning_summary_part.added`.

ResponseReasoningSummaryPartDoneEvent object { item\_id, output\_index, part, 3 more }

Emitted when a reasoning summary part is completed.

item\_id: string

The ID of the item this summary part is associated with.

output\_index: number

The index of the output item this summary part is associated with.

part: object { text, type }

The completed summary part.

text: string

The text of the summary part.

type: "summary\_text"

The type of the summary part. Always `summary_text`.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_part.done"

The type of the event. Always `response.reasoning_summary_part.done`.

ResponseReasoningSummaryTextDeltaEvent object { delta, item\_id, output\_index, 3 more }

Emitted when a delta is added to a reasoning summary text.

delta: string

The text delta that was added to the summary.

item\_id: string

The ID of the item this summary text delta is associated with.

output\_index: number

The index of the output item this summary text delta is associated with.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_text.delta"

The type of the event. Always `response.reasoning_summary_text.delta`.

ResponseReasoningSummaryTextDoneEvent object { item\_id, output\_index, sequence\_number, 3 more }

Emitted when a reasoning summary text is completed.

item\_id: string

The ID of the item this summary text is associated with.

output\_index: number

The index of the output item this summary text is associated with.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

text: string

The full text of the completed reasoning summary.

type: "response.reasoning\_summary\_text.done"

The type of the event. Always `response.reasoning_summary_text.done`.

ResponseReasoningTextDeltaEvent object { content\_index, delta, item\_id, 3 more }

Emitted when a delta is added to a reasoning text.

content\_index: number

The index of the reasoning content part this delta is associated with.

delta: string

The text delta that was added to the reasoning content.

item\_id: string

The ID of the item this reasoning text delta is associated with.

output\_index: number

The index of the output item this reasoning text delta is associated with.

sequence\_number: number

The sequence number of this event.

type: "response.reasoning\_text.delta"

The type of the event. Always `response.reasoning_text.delta`.

ResponseReasoningTextDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a reasoning text is completed.

content\_index: number

The index of the reasoning content part.

item\_id: string

The ID of the item this reasoning text is associated with.

output\_index: number

The index of the output item this reasoning text is associated with.

sequence\_number: number

The sequence number of this event.

text: string

The full text of the completed reasoning content.

type: "response.reasoning\_text.done"

The type of the event. Always `response.reasoning_text.done`.

ResponseRefusalDeltaEvent object { content\_index, delta, item\_id, 3 more }

Emitted when there is a partial refusal text.

content\_index: number

The index of the content part that the refusal text is added to.

delta: string

The refusal text that is added.

item\_id: string

The ID of the output item that the refusal text is added to.

output\_index: number

The index of the output item that the refusal text is added to.

sequence\_number: number

The sequence number of this event.

type: "response.refusal.delta"

The type of the event. Always `response.refusal.delta`.

ResponseRefusalDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when refusal text is finalized.

content\_index: number

The index of the content part that the refusal text is finalized.

item\_id: string

The ID of the output item that the refusal text is finalized.

output\_index: number

The index of the output item that the refusal text is finalized.

refusal: string

The refusal text that is finalized.

sequence\_number: number

The sequence number of this event.

type: "response.refusal.done"

The type of the event. Always `response.refusal.done`.

ResponseStatus = "completed" or "failed" or "in\_progress" or 3 more

The status of the response generation. One of `completed`, `failed`, `in_progress`, `cancelled`, `queued`, or `incomplete`.

ResponseStreamEvent = [ResponseAudioDeltaEvent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_audio_delta_event%20%3E%20\(schema\)) { delta, sequence\_number, type } or [ResponseAudioDoneEvent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_audio_done_event%20%3E%20\(schema\)) { sequence\_number, type } or [ResponseAudioTranscriptDeltaEvent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_audio_transcript_delta_event%20%3E%20\(schema\)) { delta, sequence\_number, type } or 50 more

Emitted when there is a partial audio response.

One of the following:

ResponseAudioDeltaEvent object { delta, sequence\_number, type }

Emitted when there is a partial audio response.

delta: string

A chunk of Base64 encoded response audio bytes.

sequence\_number: number

A sequence number for this chunk of the stream response.

type: "response.audio.delta"

The type of the event. Always `response.audio.delta`.

ResponseAudioDoneEvent object { sequence\_number, type }

Emitted when the audio response is complete.

sequence\_number: number

The sequence number of the delta.

type: "response.audio.done"

The type of the event. Always `response.audio.done`.

ResponseAudioTranscriptDeltaEvent object { delta, sequence\_number, type }

Emitted when there is a partial transcript of audio.

delta: string

The partial transcript of the audio response.

sequence\_number: number

The sequence number of this event.

type: "response.audio.transcript.delta"

The type of the event. Always `response.audio.transcript.delta`.

ResponseAudioTranscriptDoneEvent object { sequence\_number, type }

Emitted when the full audio transcript is completed.

sequence\_number: number

The sequence number of this event.

type: "response.audio.transcript.done"

The type of the event. Always `response.audio.transcript.done`.

ResponseCodeInterpreterCallCodeDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when a partial code snippet is streamed by the code interpreter.

delta: string

The partial code snippet being streamed by the code interpreter.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code is being streamed.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call\_code.delta"

The type of the event. Always `response.code_interpreter_call_code.delta`.

ResponseCodeInterpreterCallCodeDoneEvent object { code, item\_id, output\_index, 2 more }

Emitted when the code snippet is finalized by the code interpreter.

code: string

The final code snippet output by the code interpreter.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code is finalized.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call\_code.done"

The type of the event. Always `response.code_interpreter_call_code.done`.

ResponseCodeInterpreterCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the code interpreter call is completed.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter call is completed.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.completed"

The type of the event. Always `response.code_interpreter_call.completed`.

ResponseCodeInterpreterCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when a code interpreter call is in progress.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter call is in progress.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.in\_progress"

The type of the event. Always `response.code_interpreter_call.in_progress`.

ResponseCodeInterpreterCallInterpretingEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the code interpreter is actively interpreting the code snippet.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter is interpreting code.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.interpreting"

The type of the event. Always `response.code_interpreter_call.interpreting`.

ResponseCompletedEvent object { response, sequence\_number, type }

Emitted when the model response is complete.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

Properties of the completed response.

sequence\_number: number

The sequence number for this event.

type: "response.completed"

The type of the event. Always `response.completed`.

ResponseContentPartAddedEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a new content part is added.

content\_index: number

The index of the content part that was added.

item\_id: string

The ID of the output item that the content part was added to.

output\_index: number

The index of the output item that the content part was added to.

part: [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type } or object { text, type }

The content part that was added.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

sequence\_number: number

The sequence number of this event.

type: "response.content\_part.added"

The type of the event. Always `response.content_part.added`.

ResponseContentPartDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a content part is done.

content\_index: number

The index of the content part that is done.

item\_id: string

The ID of the output item that the content part was added to.

output\_index: number

The index of the output item that the content part was added to.

part: [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type } or object { text, type }

The content part that is done.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

sequence\_number: number

The sequence number of this event.

type: "response.content\_part.done"

The type of the event. Always `response.content_part.done`.

ResponseCreatedEvent object { response, sequence\_number, type }

An event that is emitted when a response is created.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that was created.

sequence\_number: number

The sequence number for this event.

type: "response.created"

The type of the event. Always `response.created`.

ResponseErrorEvent object { code, message, param, 2 more }

Emitted when an error occurs.

code: string

The error code.

message: string

The error message.

param: string

The error parameter.

sequence\_number: number

The sequence number of this event.

type: "error"

The type of the event. Always `error`.

ResponseFunctionCallArgumentsDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when there is a partial function-call arguments delta.

delta: string

The function-call arguments delta that is added.

item\_id: string

The ID of the output item that the function-call arguments delta is added to.

output\_index: number

The index of the output item that the function-call arguments delta is added to.

sequence\_number: number

The sequence number of this event.

type: "response.function\_call\_arguments.delta"

The type of the event. Always `response.function_call_arguments.delta`.

ResponseFunctionCallArgumentsDoneEvent object { arguments, item\_id, name, 3 more }

Emitted when function-call arguments are finalized.

arguments: string

The function-call arguments.

item\_id: string

The ID of the item.

name: string

The name of the function that was called.

output\_index: number

The index of the output item.

sequence\_number: number

The sequence number of this event.

type: "response.function\_call\_arguments.done"

ResponseInProgressEvent object { response, sequence\_number, type }

Emitted when the response is in progress.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that is in progress.

sequence\_number: number

The sequence number of this event.

type: "response.in\_progress"

The type of the event. Always `response.in_progress`.

ResponseFailedEvent object { response, sequence\_number, type }

An event that is emitted when a response fails.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that failed.

sequence\_number: number

The sequence number of this event.

type: "response.failed"

The type of the event. Always `response.failed`.

ResponseIncompleteEvent object { response, sequence\_number, type }

An event that is emitted when a response finishes as incomplete.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that was incomplete.

sequence\_number: number

The sequence number of this event.

type: "response.incomplete"

The type of the event. Always `response.incomplete`.

ResponseOutputItemAddedEvent object { item, output\_index, sequence\_number, type }

Emitted when a new output item is added.

item: [ResponseOutputItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_item%20%3E%20\(schema\))

The output item that was added.

output\_index: number

The index of the output item that was added.

sequence\_number: number

The sequence number of this event.

type: "response.output\_item.added"

The type of the event. Always `response.output_item.added`.

ResponseOutputItemDoneEvent object { item, output\_index, sequence\_number, type }

Emitted when an output item is marked done.

item: [ResponseOutputItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_item%20%3E%20\(schema\))

The output item that was marked done.

output\_index: number

The index of the output item that was marked done.

sequence\_number: number

The sequence number of this event.

type: "response.output\_item.done"

The type of the event. Always `response.output_item.done`.

ResponseReasoningSummaryPartAddedEvent object { item\_id, output\_index, part, 3 more }

Emitted when a new reasoning summary part is added.

item\_id: string

The ID of the item this summary part is associated with.

output\_index: number

The index of the output item this summary part is associated with.

part: object { text, type }

The summary part that was added.

text: string

The text of the summary part.

type: "summary\_text"

The type of the summary part. Always `summary_text`.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_part.added"

The type of the event. Always `response.reasoning_summary_part.added`.

ResponseReasoningSummaryPartDoneEvent object { item\_id, output\_index, part, 3 more }

Emitted when a reasoning summary part is completed.

item\_id: string

The ID of the item this summary part is associated with.

output\_index: number

The index of the output item this summary part is associated with.

part: object { text, type }

The completed summary part.

text: string

The text of the summary part.

type: "summary\_text"

The type of the summary part. Always `summary_text`.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_part.done"

The type of the event. Always `response.reasoning_summary_part.done`.

ResponseReasoningSummaryTextDeltaEvent object { delta, item\_id, output\_index, 3 more }

Emitted when a delta is added to a reasoning summary text.

delta: string

The text delta that was added to the summary.

item\_id: string

The ID of the item this summary text delta is associated with.

output\_index: number

The index of the output item this summary text delta is associated with.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_text.delta"

The type of the event. Always `response.reasoning_summary_text.delta`.

ResponseReasoningSummaryTextDoneEvent object { item\_id, output\_index, sequence\_number, 3 more }

Emitted when a reasoning summary text is completed.

item\_id: string

The ID of the item this summary text is associated with.

output\_index: number

The index of the output item this summary text is associated with.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

text: string

The full text of the completed reasoning summary.

type: "response.reasoning\_summary\_text.done"

The type of the event. Always `response.reasoning_summary_text.done`.

ResponseReasoningTextDeltaEvent object { content\_index, delta, item\_id, 3 more }

Emitted when a delta is added to a reasoning text.

content\_index: number

The index of the reasoning content part this delta is associated with.

delta: string

The text delta that was added to the reasoning content.

item\_id: string

The ID of the item this reasoning text delta is associated with.

output\_index: number

The index of the output item this reasoning text delta is associated with.

sequence\_number: number

The sequence number of this event.

type: "response.reasoning\_text.delta"

The type of the event. Always `response.reasoning_text.delta`.

ResponseReasoningTextDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a reasoning text is completed.

content\_index: number

The index of the reasoning content part.

item\_id: string

The ID of the item this reasoning text is associated with.

output\_index: number

The index of the output item this reasoning text is associated with.

sequence\_number: number

The sequence number of this event.

text: string

The full text of the completed reasoning content.

type: "response.reasoning\_text.done"

The type of the event. Always `response.reasoning_text.done`.

ResponseRefusalDeltaEvent object { content\_index, delta, item\_id, 3 more }

Emitted when there is a partial refusal text.

content\_index: number

The index of the content part that the refusal text is added to.

delta: string

The refusal text that is added.

item\_id: string

The ID of the output item that the refusal text is added to.

output\_index: number

The index of the output item that the refusal text is added to.

sequence\_number: number

The sequence number of this event.

type: "response.refusal.delta"

The type of the event. Always `response.refusal.delta`.

ResponseRefusalDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when refusal text is finalized.

content\_index: number

The index of the content part that the refusal text is finalized.

item\_id: string

The ID of the output item that the refusal text is finalized.

output\_index: number

The index of the output item that the refusal text is finalized.

refusal: string

The refusal text that is finalized.

sequence\_number: number

The sequence number of this event.

type: "response.refusal.done"

The type of the event. Always `response.refusal.done`.

ResponseTextDeltaEvent object { content\_index, delta, item\_id, 4 more }

Emitted when there is an additional text delta.

content\_index: number

The index of the content part that the text delta was added to.

delta: string

The text delta that was added.

item\_id: string

The ID of the output item that the text delta was added to.

logprobs: array of object { token, logprob, top\_logprobs }

The log probabilities of the tokens in the delta.

token: string

A possible text token.

logprob: number

The log probability of this token.

top\_logprobs: optional array of object { token, logprob }

The log probabilities of up to 20 of the most likely tokens.

token: optional string

A possible text token.

logprob: optional number

The log probability of this token.

output\_index: number

The index of the output item that the text delta was added to.

sequence\_number: number

The sequence number for this event.

type: "response.output\_text.delta"

The type of the event. Always `response.output_text.delta`.

ResponseTextDoneEvent object { content\_index, item\_id, logprobs, 4 more }

Emitted when text content is finalized.

content\_index: number

The index of the content part that the text content is finalized.

item\_id: string

The ID of the output item that the text content is finalized.

logprobs: array of object { token, logprob, top\_logprobs }

The log probabilities of the tokens in the delta.

token: string

A possible text token.

logprob: number

The log probability of this token.

top\_logprobs: optional array of object { token, logprob }

The log probabilities of up to 20 of the most likely tokens.

token: optional string

A possible text token.

logprob: optional number

The log probability of this token.

output\_index: number

The index of the output item that the text content is finalized.

sequence\_number: number

The sequence number for this event.

text: string

The text content that is finalized.

type: "response.output\_text.done"

The type of the event. Always `response.output_text.done`.

ResponseImageGenCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call has completed and the final image is available.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.image\_generation\_call.completed"

The type of the event. Always ‘response.image\_generation\_call.completed’.

ResponseImageGenCallGeneratingEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call is actively generating an image (intermediate state).

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.generating"

The type of the event. Always ‘response.image\_generation\_call.generating’.

ResponseImageGenCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call is in progress.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.in\_progress"

The type of the event. Always ‘response.image\_generation\_call.in\_progress’.

ResponseImageGenCallPartialImageEvent object { item\_id, output\_index, partial\_image\_b64, 3 more }

Emitted when a partial image is available during image generation streaming.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

partial\_image\_b64: string

Base64-encoded partial image data, suitable for rendering as an image.

partial\_image\_index: number

0-based index for the partial image (backend is 1-based, but this is 0-based for the user).

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.partial\_image"

The type of the event. Always ‘response.image\_generation\_call.partial\_image’.

ResponseMcpCallArgumentsDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when there is a delta (partial update) to the arguments of an MCP tool call.

delta: string

A JSON string containing the partial update to the arguments for the MCP tool call.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call\_arguments.delta"

The type of the event. Always ‘response.mcp\_call\_arguments.delta’.

ResponseMcpCallArgumentsDoneEvent object { arguments, item\_id, output\_index, 2 more }

Emitted when the arguments for an MCP tool call are finalized.

arguments: string

A JSON string containing the finalized arguments for the MCP tool call.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call\_arguments.done"

The type of the event. Always ‘response.mcp\_call\_arguments.done’.

ResponseMcpCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call has completed successfully.

item\_id: string

The ID of the MCP tool call item that completed.

output\_index: number

The index of the output item that completed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.completed"

The type of the event. Always ‘response.mcp\_call.completed’.

ResponseMcpCallFailedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call has failed.

item\_id: string

The ID of the MCP tool call item that failed.

output\_index: number

The index of the output item that failed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.failed"

The type of the event. Always ‘response.mcp\_call.failed’.

ResponseMcpCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call is in progress.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.in\_progress"

The type of the event. Always ‘response.mcp\_call.in\_progress’.

ResponseMcpListToolsCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the list of available MCP tools has been successfully retrieved.

item\_id: string

The ID of the MCP tool call item that produced this output.

output\_index: number

The index of the output item that was processed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.completed"

The type of the event. Always ‘response.mcp\_list\_tools.completed’.

ResponseMcpListToolsFailedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the attempt to list available MCP tools has failed.

item\_id: string

The ID of the MCP tool call item that failed.

output\_index: number

The index of the output item that failed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.failed"

The type of the event. Always ‘response.mcp\_list\_tools.failed’.

ResponseMcpListToolsInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the system is in the process of retrieving the list of available MCP tools.

item\_id: string

The ID of the MCP tool call item that is being processed.

output\_index: number

The index of the output item that is being processed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.in\_progress"

The type of the event. Always ‘response.mcp\_list\_tools.in\_progress’.

ResponseOutputTextAnnotationAddedEvent object { annotation, annotation\_index, content\_index, 4 more }

Emitted when an annotation is added to output text content.

annotation: unknown

The annotation object being added. (See annotation schema for details.)

annotation\_index: number

The index of the annotation within the content part.

content\_index: number

The index of the content part within the output item.

item\_id: string

The unique identifier of the item to which the annotation is being added.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.output\_text.annotation.added"

The type of the event. Always ‘response.output\_text.annotation.added’.

ResponseQueuedEvent object { response, sequence\_number, type }

Emitted when a response is queued and waiting to be processed.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The full response object that is queued.

sequence\_number: number

The sequence number for this event.

type: "response.queued"

The type of the event. Always ‘response.queued’.

ResponseCustomToolCallInputDeltaEvent object { delta, item\_id, output\_index, 2 more }

Event representing a delta (partial update) to the input of a custom tool call.

delta: string

The incremental input data (delta) for the custom tool call.

item\_id: string

Unique identifier for the API item associated with this event.

output\_index: number

The index of the output this delta applies to.

sequence\_number: number

The sequence number of this event.

type: "response.custom\_tool\_call\_input.delta"

The event type identifier.

ResponseCustomToolCallInputDoneEvent object { input, item\_id, output\_index, 2 more }

Event indicating that input for a custom tool call is complete.

input: string

The complete input data for the custom tool call.

item\_id: string

Unique identifier for the API item associated with this event.

output\_index: number

The index of the output this event applies to.

sequence\_number: number

The sequence number of this event.

type: "response.custom\_tool\_call\_input.done"

The event type identifier.

ResponseTextConfig object { format, verbosity }

Configuration options for a text response from the model. Can be plain text or structured JSON data. Learn more:

- [Text inputs and outputs](https://developers.openai.com/docs/guides/text)
- [Structured Outputs](https://developers.openai.com/docs/guides/structured-outputs)

format: optional [ResponseFormatTextConfig](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_format_text_config%20%3E%20\(schema\))

An object specifying the format that the model must output.

Configuring `{ "type": "json_schema" }` enables Structured Outputs, which ensures the model will match your supplied JSON schema. Learn more in the [Structured Outputs guide](https://developers.openai.com/docs/guides/structured-outputs).

The default format is `{ "type": "text" }` with no additional options.

**Not recommended for gpt-4o and newer models:**

Setting to `{ "type": "json_object" }` enables the older JSON mode, which ensures the message the model generates is valid JSON. Using `json_schema` is preferred for models that support it.

verbosity: optional "low" or "medium" or "high"

Constrains the verbosity of the model’s response. Lower values will result in more concise responses, while higher values will result in more verbose responses. Currently supported values are `low`, `medium`, and `high`.

ResponseTextDeltaEvent object { content\_index, delta, item\_id, 4 more }

Emitted when there is an additional text delta.

content\_index: number

The index of the content part that the text delta was added to.

delta: string

The text delta that was added.

item\_id: string

The ID of the output item that the text delta was added to.

logprobs: array of object { token, logprob, top\_logprobs }

The log probabilities of the tokens in the delta.

token: string

A possible text token.

logprob: number

The log probability of this token.

top\_logprobs: optional array of object { token, logprob }

The log probabilities of up to 20 of the most likely tokens.

token: optional string

A possible text token.

logprob: optional number

The log probability of this token.

output\_index: number

The index of the output item that the text delta was added to.

sequence\_number: number

The sequence number for this event.

type: "response.output\_text.delta"

The type of the event. Always `response.output_text.delta`.

ResponseTextDoneEvent object { content\_index, item\_id, logprobs, 4 more }

Emitted when text content is finalized.

content\_index: number

The index of the content part that the text content is finalized.

item\_id: string

The ID of the output item that the text content is finalized.

logprobs: array of object { token, logprob, top\_logprobs }

The log probabilities of the tokens in the delta.

token: string

A possible text token.

logprob: number

The log probability of this token.

top\_logprobs: optional array of object { token, logprob }

The log probabilities of up to 20 of the most likely tokens.

token: optional string

A possible text token.

logprob: optional number

The log probability of this token.

output\_index: number

The index of the output item that the text content is finalized.

sequence\_number: number

The sequence number for this event.

text: string

The text content that is finalized.

type: "response.output\_text.done"

The type of the event. Always `response.output_text.done`.

ResponseUsage object { input\_tokens, input\_tokens\_details, output\_tokens, 2 more }

Represents token usage details including input tokens, output tokens, a breakdown of output tokens, and the total tokens used.

input\_tokens: number

The number of input tokens.

input\_tokens\_details: object { cached\_tokens }

A detailed breakdown of the input tokens.

cached\_tokens: number

The number of tokens that were retrieved from the cache. [More on prompt caching](https://developers.openai.com/docs/guides/prompt-caching).

output\_tokens: number

The number of output tokens.

output\_tokens\_details: object { reasoning\_tokens }

A detailed breakdown of the output tokens.

reasoning\_tokens: number

The number of reasoning tokens.

total\_tokens: number

The total number of tokens used.

ResponsesClientEvent object { type, background, context\_management, 27 more }

type: "response.create"

The type of the client event. Always `response.create`.

background: optional boolean

Whether to run the model response in the background. [Learn more](https://developers.openai.com/docs/guides/background).

context\_management: optional array of object { type, compact\_threshold }

Context management configuration for this request.

type: string

The context management entry type. Currently only ‘compaction’ is supported.

compact\_threshold: optional number

Token threshold at which compaction should be triggered for this entry.

minimum1000

conversation: optional string or [ResponseConversationParam](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_conversation_param%20%3E%20\(schema\)) { id }

The conversation that this response belongs to. Items from this conversation are prepended to `input_items` for this response request. Input items and output items from this response are automatically added to this conversation after this response completes.

One of the following:

ConversationID = string

The unique ID of the conversation.

ResponseConversationParam object { id }

The conversation that this response belongs to.

id: string

The unique ID of the conversation.

include: optional array of [ResponseIncludable](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_includable%20%3E%20\(schema\))

Specify additional output data to include in the model response. Currently supported values are:

- `web_search_call.action.sources`: Include the sources of the web search tool call.
- `code_interpreter_call.outputs`: Includes the outputs of python code execution in code interpreter tool call items.
- `computer_call_output.output.image_url`: Include image urls from the computer call output.
- `file_search_call.results`: Include the search results of the file search tool call.
- `message.input_image.image_url`: Include image urls from the input message.
- `message.output_text.logprobs`: Include logprobs with assistant messages.
- `reasoning.encrypted_content`: Includes an encrypted version of reasoning tokens in reasoning item outputs. This enables reasoning items to be used in multi-turn conversations when using the Responses API statelessly (like when the `store` parameter is set to `false`, or when an organization is enrolled in the zero data retention program).

input: optional string or array of [EasyInputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20easy_input_message%20%3E%20\(schema\)) { content, role, phase, type } or object { content, role, status, type } or [ResponseOutputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_message%20%3E%20\(schema\)) { id, content, role, 3 more } or 25 more

One of the following:

TextInput = string

A text input to the model, equivalent to a text input with the `user` role.

InputItemList = array of [EasyInputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20easy_input_message%20%3E%20\(schema\)) { content, role, phase, type } or object { content, role, status, type } or [ResponseOutputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_message%20%3E%20\(schema\)) { id, content, role, 3 more } or 25 more

A list of one or many input items to the model, containing different content types.

One of the following:

EasyInputMessage object { content, role, phase, type }

A message input to the model with a role indicating instruction following hierarchy. Instructions given with the `developer` or `system` role take precedence over instructions given with the `user` role. Messages with the `assistant` role are presumed to have been generated by the model in previous interactions.

content: string or [ResponseInputMessageContentList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_content_list%20%3E%20\(schema\)) {,, }

Text, image, or audio input to the model, used to generate a response. Can also contain previous assistant responses.

One of the following:

TextInput = string

A text input to the model.

ResponseInputMessageContentList = array of [ResponseInputContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_content%20%3E%20\(schema\))

A list of one or many input items to the model, containing different content types.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

role: "user" or "assistant" or "system" or "developer"

The role of the message input. One of `user`, `assistant`, `system`, or `developer`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

type: optional "message"

The type of the message input. Always `message`.

Message object { content, role, status, type }

A message input to the model with a role indicating instruction following hierarchy. Instructions given with the `developer` or `system` role take precedence over instructions given with the `user` role.

content: [ResponseInputMessageContentList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_content_list%20%3E%20\(schema\)) {,, }

A list of one or many input items to the model, containing different content types.

role: "user" or "system" or "developer"

The role of the message input. One of `user`, `system`, or `developer`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: optional "message"

The type of the message input. Always set to `message`.

ResponseOutputMessage object { id, content, role, 3 more }

An output message from the model.

id: string

The unique ID of the output message.

content: array of [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type }

The content of the output message.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

role: "assistant"

The role of the output message. Always `assistant`.

status: "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "message"

The type of the output message. Always `message`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

FileSearchCall object { id, queries, status, 2 more }

The results of a file search tool call. See the [file search guide](https://developers.openai.com/docs/guides/tools-file-search) for more information.

id: string

The unique ID of the file search tool call.

queries: array of string

The queries used to search for files.

status: "in\_progress" or "searching" or "completed" or 2 more

The status of the file search tool call. One of `in_progress`, `searching`, `incomplete` or `failed`,

type: "file\_search\_call"

The type of the file search tool call. Always `file_search_call`.

results: optional array of object { attributes, file\_id, filename, 2 more }

The results of the file search tool call.

attributes: optional map\[string or number or boolean\]

Set of 16 key-value pairs that can be attached to an object. This can be useful for storing additional information about the object in a structured format, and querying for objects via API or the dashboard. Keys are strings with a maximum length of 64 characters. Values are strings with a maximum length of 512 characters, booleans, or numbers.

file\_id: optional string

The unique ID of the file.

filename: optional string

The name of the file.

score: optional number

The relevance score of the file - a value between 0 and 1.

formatfloat

text: optional string

The text that was retrieved from the file.

ComputerCall object { id, call\_id, pending\_safety\_checks, 4 more }

A tool call to a computer use tool. See the [computer use guide](https://developers.openai.com/docs/guides/tools-computer-use) for more information.

id: string

The unique ID of the computer call.

call\_id: string

An identifier used when responding to the tool call with output.

pending\_safety\_checks: array of object { id, code, message }

The pending safety checks for the computer call.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "computer\_call"

The type of the computer call. Always `computer_call`.

action: optional [ComputerAction](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action%20%3E%20\(schema\))

A click action.

actions: optional [ComputerActionList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action_list%20%3E%20\(schema\)) { Click, DoubleClick, Drag, 6 more }

Flattened batched actions for `computer_use`. Each action includes an `type` discriminator and action-specific fields.

ComputerCallOutput object { call\_id, output, type, 3 more }

The output of a computer tool call.

call\_id: string

The ID of the computer tool call that produced the output.

maxLength64

minLength1

output: [ResponseComputerToolCallOutputScreenshot](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_computer_tool_call_output_screenshot%20%3E%20\(schema\)) { type, file\_id, image\_url }

A computer screenshot image used with the computer use tool.

type: "computer\_call\_output"

The type of the computer tool call output. Always `computer_call_output`.

id: optional string

The ID of the computer tool call output.

acknowledged\_safety\_checks: optional array of object { id, code, message }

The safety checks reported by the API that have been acknowledged by the developer.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

WebSearchCall object { id, action, status, type }

The results of a web search tool call. See the [web search guide](https://developers.openai.com/docs/guides/tools-web-search) for more information.

id: string

The unique ID of the web search tool call.

action: object { query, type, queries, sources } or object { type, url } or object { pattern, type, url }

An object describing the specific action taken in this web search call. Includes details on how the model used the web (search, open\_page, find\_in\_page).

One of the following:

Search object { query, type, queries, sources }

Action type “search” - Performs a web search query.

query: string

\[DEPRECATED\] The search query.

type: "search"

The action type.

queries: optional array of string

The search queries.

sources: optional array of object { type, url }

The sources used in the search.

type: "url"

The type of source. Always `url`.

url: string

The URL of the source.

formaturi

OpenPage object { type, url }

Action type “open\_page” - Opens a specific URL from search results.

type: "open\_page"

The action type.

url: optional string

The URL opened by the model.

formaturi

FindInPage object { pattern, type, url }

Action type “find\_in\_page”: Searches for a pattern within a loaded page.

pattern: string

The pattern or text to search for within the page.

type: "find\_in\_page"

The action type.

url: string

The URL of the page searched for the pattern.

formaturi

status: "in\_progress" or "searching" or "completed" or "failed"

The status of the web search tool call.

type: "web\_search\_call"

The type of the web search tool call. Always `web_search_call`.

FunctionCall object { arguments, call\_id, name, 4 more }

A tool call to run a function. See the [function calling guide](https://developers.openai.com/docs/guides/function-calling) for more information.

arguments: string

A JSON string of the arguments to pass to the function.

call\_id: string

The unique ID of the function tool call generated by the model.

name: string

The name of the function to run.

type: "function\_call"

The type of the function tool call. Always `function_call`.

id: optional string

The unique ID of the function tool call.

namespace: optional string

The namespace of the function to run.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

FunctionCallOutput object { call\_id, output, type, 2 more }

The output of a function tool call.

call\_id: string

The unique ID of the function tool call generated by the model.

maxLength64

minLength1

output: string or array of [ResponseInputTextContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text_content%20%3E%20\(schema\)) { text, type } or [ResponseInputImageContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image_content%20%3E%20\(schema\)) { type, detail, file\_id, image\_url } or [ResponseInputFileContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file_content%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the function tool call.

One of the following:

string

A JSON string of the output of the function tool call.

array of [ResponseInputTextContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text_content%20%3E%20\(schema\)) { text, type } or [ResponseInputImageContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image_content%20%3E%20\(schema\)) { type, detail, file\_id, image\_url } or [ResponseInputFileContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file_content%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

An array of content outputs (text, image, file) for the function tool call.

One of the following:

ResponseInputTextContent object { text, type }

A text input to the model.

text: string

The text input to the model.

maxLength10485760

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImageContent object { type, detail, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision)

type: "input\_image"

The type of the input item. Always `input_image`.

detail: optional "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

maxLength20971520

formaturi

ResponseInputFileContent object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The base64-encoded data of the file to be sent to the model.

maxLength73400320

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

type: "function\_call\_output"

The type of the function tool call output. Always `function_call_output`.

id: optional string

The unique ID of the function tool call output. Populated when this item is returned via API.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

ToolSearchCall object { arguments, type, id, 3 more }

arguments: unknown

The arguments supplied to the tool search call.

type: "tool\_search\_call"

The item type. Always `tool_search_call`.

id: optional string

The unique ID of this tool search call.

call\_id: optional string

The unique ID of the tool search call generated by the model.

maxLength64

minLength1

execution: optional "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: optional "in\_progress" or "completed" or "incomplete"

The status of the tool search call.

ToolSearchOutput object { tools, type, id, 3 more }

tools: array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

The loaded tool definitions returned by the tool search output.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

type: "tool\_search\_output"

The item type. Always `tool_search_output`.

id: optional string

The unique ID of this tool search output.

call\_id: optional string

The unique ID of the tool search call generated by the model.

maxLength64

minLength1

execution: optional "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: optional "in\_progress" or "completed" or "incomplete"

The status of the tool search output.

Reasoning object { id, summary, type, 3 more }

A description of the chain of thought used by a reasoning model while generating a response. Be sure to include these items in your `input` to the Responses API for subsequent turns of a conversation if you are manually [managing context](https://developers.openai.com/docs/guides/conversation-state).

id: string

The unique identifier of the reasoning content.

summary: array of [SummaryTextContent](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20summary_text_content%20%3E%20\(schema\)) { text, type }

Reasoning summary content.

text: string

A summary of the reasoning output from the model so far.

type: "summary\_text"

The type of the object. Always `summary_text`.

type: "reasoning"

The type of the object. Always `reasoning`.

content: optional array of object { text, type }

Reasoning text content.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

encrypted\_content: optional string

The encrypted content of the reasoning item - populated when a response is generated with `reasoning.encrypted_content` in the `include` parameter.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

Compaction object { encrypted\_content, type, id }

encrypted\_content: string

The encrypted content of the compaction summary.

maxLength10485760

type: "compaction"

The type of the item. Always `compaction`.

id: optional string

The ID of the compaction item.

ImageGenerationCall object { id, result, status, type }

An image generation request made by the model.

id: string

The unique ID of the image generation call.

result: string

The generated image encoded in base64.

status: "in\_progress" or "completed" or "generating" or "failed"

The status of the image generation call.

type: "image\_generation\_call"

The type of the image generation call. Always `image_generation_call`.

CodeInterpreterCall object { id, code, container\_id, 3 more }

A tool call to run code.

id: string

The unique ID of the code interpreter tool call.

code: string

The code to run, or null if not available.

container\_id: string

The ID of the container used to run the code.

outputs: array of object { logs, type } or object { type, url }

The outputs generated by the code interpreter, such as logs or images. Can be null if no outputs are available.

One of the following:

Logs object { logs, type }

The logs output from the code interpreter.

logs: string

The logs output from the code interpreter.

type: "logs"

The type of the output. Always `logs`.

Image object { type, url }

The image output from the code interpreter.

type: "image"

The type of the output. Always `image`.

url: string

The URL of the image output from the code interpreter.

formaturi

status: "in\_progress" or "completed" or "incomplete" or 2 more

The status of the code interpreter tool call. Valid values are `in_progress`, `completed`, `incomplete`, `interpreting`, and `failed`.

type: "code\_interpreter\_call"

The type of the code interpreter tool call. Always `code_interpreter_call`.

LocalShellCall object { id, action, call\_id, 2 more }

A tool call to run a command on the local shell.

id: string

The unique ID of the local shell call.

action: object { command, env, type, 3 more }

Execute a shell command on the server.

command: array of string

The command to run.

env: map\[string\]

Environment variables to set for the command.

type: "exec"

The type of the local shell action. Always `exec`.

timeout\_ms: optional number

Optional timeout in milliseconds for the command.

user: optional string

Optional user to run the command as.

working\_directory: optional string

Optional working directory to run the command in.

call\_id: string

The unique ID of the local shell tool call generated by the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the local shell call.

type: "local\_shell\_call"

The type of the local shell call. Always `local_shell_call`.

LocalShellCallOutput object { id, output, type, status }

The output of a local shell tool call.

id: string

The unique ID of the local shell tool call generated by the model.

output: string

A JSON string of the output of the local shell tool call.

type: "local\_shell\_call\_output"

The type of the local shell tool call output. Always `local_shell_call_output`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`.

ShellCall object { action, call\_id, type, 3 more }

A tool representing a request to execute one or more shell commands.

action: object { commands, max\_output\_length, timeout\_ms }

The shell commands and limits that describe how to run the tool call.

commands: array of string

Ordered shell commands for the execution environment to run.

max\_output\_length: optional number

Maximum number of UTF-8 characters to capture from combined stdout and stderr output.

timeout\_ms: optional number

Maximum wall-clock time in milliseconds to allow the shell commands to run.

call\_id: string

The unique ID of the shell tool call generated by the model.

maxLength64

minLength1

type: "shell\_call"

The type of the item. Always `shell_call`.

id: optional string

The unique ID of the shell tool call. Populated when this item is returned via API.

environment: optional [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

The environment to execute the shell commands in.

One of the following:

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

status: optional "in\_progress" or "completed" or "incomplete"

The status of the shell call. One of `in_progress`, `completed`, or `incomplete`.

ShellCallOutput object { call\_id, output, type, 3 more }

The streamed output items emitted by a shell tool call.

call\_id: string

The unique ID of the shell tool call generated by the model.

maxLength64

minLength1

output: array of [ResponseFunctionShellCallOutputContent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_function_shell_call_output_content%20%3E%20\(schema\)) { outcome, stderr, stdout }

Captured chunks of stdout and stderr output, along with their associated outcomes.

outcome: object { type } or object { exit\_code, type }

The exit or timeout outcome associated with this shell call.

One of the following:

Timeout object { type }

Indicates that the shell call exceeded its configured time limit.

type: "timeout"

The outcome type. Always `timeout`.

Exit object { exit\_code, type }

Indicates that the shell commands finished and returned an exit code.

exit\_code: number

The exit code returned by the shell process.

type: "exit"

The outcome type. Always `exit`.

stderr: string

Captured stderr output for the shell call.

maxLength10485760

stdout: string

Captured stdout output for the shell call.

maxLength10485760

type: "shell\_call\_output"

The type of the item. Always `shell_call_output`.

id: optional string

The unique ID of the shell tool call output. Populated when this item is returned via API.

max\_output\_length: optional number

The maximum number of UTF-8 characters captured for this shell call’s combined output.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the shell call output.

ApplyPatchCall object { call\_id, operation, status, 2 more }

A tool call representing a request to create, delete, or update files using diff patches.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

maxLength64

minLength1

operation: object { diff, path, type } or object { path, type } or object { diff, path, type }

The specific create, delete, or update instruction for the apply\_patch tool call.

One of the following:

CreateFile object { diff, path, type }

Instruction for creating a new file via the apply\_patch tool.

diff: string

Unified diff content to apply when creating the file.

maxLength10485760

path: string

Path of the file to create relative to the workspace root.

minLength1

type: "create\_file"

The operation type. Always `create_file`.

DeleteFile object { path, type }

Instruction for deleting an existing file via the apply\_patch tool.

path: string

Path of the file to delete relative to the workspace root.

minLength1

type: "delete\_file"

The operation type. Always `delete_file`.

UpdateFile object { diff, path, type }

Instruction for updating an existing file via the apply\_patch tool.

diff: string

Unified diff content to apply to the existing file.

maxLength10485760

path: string

Path of the file to update relative to the workspace root.

minLength1

type: "update\_file"

The operation type. Always `update_file`.

status: "in\_progress" or "completed"

The status of the apply patch tool call. One of `in_progress` or `completed`.

One of the following:

"in\_progress"

"completed"

type: "apply\_patch\_call"

The type of the item. Always `apply_patch_call`.

id: optional string

The unique ID of the apply patch tool call. Populated when this item is returned via API.

ApplyPatchCallOutput object { call\_id, status, type, 2 more }

The streamed output emitted by an apply patch tool call.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

maxLength64

minLength1

status: "completed" or "failed"

The status of the apply patch tool call output. One of `completed` or `failed`.

One of the following:

"completed"

"failed"

type: "apply\_patch\_call\_output"

The type of the item. Always `apply_patch_call_output`.

id: optional string

The unique ID of the apply patch tool call output. Populated when this item is returned via API.

output: optional string

Optional human-readable log text from the apply patch tool (e.g., patch results or errors).

maxLength10485760

McpListTools object { id, server\_label, tools, 2 more }

A list of tools available on an MCP server.

id: string

The unique ID of the list.

server\_label: string

The label of the MCP server.

tools: array of object { input\_schema, name, annotations, description }

The tools available on the server.

input\_schema: unknown

The JSON schema describing the tool’s input.

name: string

The name of the tool.

annotations: optional unknown

Additional annotations about the tool.

description: optional string

The description of the tool.

type: "mcp\_list\_tools"

The type of the item. Always `mcp_list_tools`.

error: optional string

Error message if the server could not list tools.

McpApprovalRequest object { id, arguments, name, 2 more }

A request for human approval of a tool invocation.

id: string

The unique ID of the approval request.

arguments: string

A JSON string of arguments for the tool.

name: string

The name of the tool to run.

server\_label: string

The label of the MCP server making the request.

type: "mcp\_approval\_request"

The type of the item. Always `mcp_approval_request`.

McpApprovalResponse object { approval\_request\_id, approve, type, 2 more }

A response to an MCP approval request.

approval\_request\_id: string

The ID of the approval request being answered.

approve: boolean

Whether the request was approved.

type: "mcp\_approval\_response"

The type of the item. Always `mcp_approval_response`.

id: optional string

The unique ID of the approval response

reason: optional string

Optional reason for the decision.

McpCall object { id, arguments, name, 6 more }

An invocation of a tool on an MCP server.

id: string

The unique ID of the tool call.

arguments: string

A JSON string of the arguments passed to the tool.

name: string

The name of the tool that was run.

server\_label: string

The label of the MCP server running the tool.

type: "mcp\_call"

The type of the item. Always `mcp_call`.

approval\_request\_id: optional string

Unique identifier for the MCP tool call approval request. Include this value in a subsequent `mcp_approval_response` input to approve or reject the corresponding tool call.

error: optional string

The error from the tool call, if any.

output: optional string

The output from the tool call.

status: optional "in\_progress" or "completed" or "incomplete" or 2 more

The status of the tool call. One of `in_progress`, `completed`, `incomplete`, `calling`, or `failed`.

CustomToolCallOutput object { call\_id, output, type, id }

The output of a custom tool call from your code, being sent back to the model.

call\_id: string

The call ID, used to map this custom tool call output to a custom tool call.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the custom tool call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the custom tool call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the custom tool call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

type: "custom\_tool\_call\_output"

The type of the custom tool call output. Always `custom_tool_call_output`.

id: optional string

The unique ID of the custom tool call output in the OpenAI platform.

CustomToolCall object { call\_id, input, name, 3 more }

A call to a custom tool created by the model.

call\_id: string

An identifier used to map this custom tool call to a tool call output.

input: string

The input for the custom tool call generated by the model.

name: string

The name of the custom tool being called.

type: "custom\_tool\_call"

The type of the custom tool call. Always `custom_tool_call`.

id: optional string

The unique ID of the custom tool call in the OpenAI platform.

namespace: optional string

The namespace of the custom tool being called.

ItemReference object { id, type }

An internal identifier for an item to reference.

id: string

The ID of the item to reference.

type: optional "item\_reference"

The type of item to reference. Always `item_reference`.

instructions: optional string

A system (or developer) message inserted into the model’s context.

When using along with `previous_response_id`, the instructions from a previous response will not be carried over to the next response. This makes it simple to swap out system (or developer) messages in new responses.

max\_output\_tokens: optional number

An upper bound for the number of tokens that can be generated for a response, including visible output tokens and [reasoning tokens](https://developers.openai.com/docs/guides/reasoning).

minimum16

max\_tool\_calls: optional number

The maximum number of total calls to built-in tools that can be processed in a response. This maximum number applies across all built-in tool calls, not per individual tool. Any further attempts to call a tool by the model will be ignored.

model: optional [ResponsesModel](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20responses_model%20%3E%20\(schema\))

Model ID used to generate the response, like `gpt-4o` or `o3`. OpenAI offers a wide range of models with different capabilities, performance characteristics, and price points. Refer to the [model guide](https://developers.openai.com/docs/models) to browse and compare available models.

parallel\_tool\_calls: optional boolean

Whether to allow the model to run tool calls in parallel.

previous\_response\_id: optional string

The unique ID of the previous response to the model. Use this to create multi-turn conversations. Learn more about [conversation state](https://developers.openai.com/docs/guides/conversation-state). Cannot be used in conjunction with `conversation`.

prompt: optional [ResponsePrompt](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_prompt%20%3E%20\(schema\)) { id, variables, version }

Reference to a prompt template and its variables. [Learn more](https://developers.openai.com/docs/guides/text?api-mode=responses#reusable-prompts).

prompt\_cache\_key: optional string

Used by OpenAI to cache responses for similar requests to optimize your cache hit rates. Replaces the `user` field. [Learn more](https://developers.openai.com/docs/guides/prompt-caching).

prompt\_cache\_retention: optional "in\_memory" or "24h"

The retention policy for the prompt cache. Set to `24h` to enable extended prompt caching, which keeps cached prefixes active for longer, up to a maximum of 24 hours. [Learn more](https://developers.openai.com/docs/guides/prompt-caching#prompt-cache-retention).

One of the following:

"in\_memory"

"24h"

reasoning: optional [Reasoning](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20reasoning%20%3E%20\(schema\)) { effort, generate\_summary, summary }

**gpt-5 and o-series models only**

Configuration options for [reasoning models](https://platform.openai.com/docs/guides/reasoning).

safety\_identifier: optional string

A stable identifier used to help detect users of your application that may be violating OpenAI’s usage policies. The IDs should be a string that uniquely identifies each user, with a maximum length of 64 characters. We recommend hashing their username or email address, in order to avoid sending us any identifying information. [Learn more](https://developers.openai.com/docs/guides/safety-best-practices#safety-identifiers).

maxLength64

service\_tier: optional "auto" or "default" or "flex" or 2 more

Specifies the processing type used for serving the request.

- If set to ‘auto’, then the request will be processed with the service tier configured in the Project settings. Unless otherwise configured, the Project will use ‘default’.
- If set to ‘default’, then the request will be processed with the standard pricing and performance for the selected model.
- If set to ‘ [flex](https://developers.openai.com/docs/guides/flex-processing) ’ or ‘ [priority](https://openai.com/api-priority-processing/) ’, then the request will be processed with the corresponding service tier.
- When not set, the default behavior is ‘auto’.

When the `service_tier` parameter is set, the response body will include the `service_tier` value based on the processing mode actually used to serve the request. This response value may be different from the value set in the parameter.

store: optional boolean

Whether to store the generated model response for later retrieval via API.

stream: optional boolean

If set to true, the model response data will be streamed to the client as it is generated using [server-sent events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#Event_stream_format). See the [Streaming section below](https://developers.openai.com/docs/api-reference/responses-streaming) for more information.

stream\_options: optional object { include\_obfuscation }

Options for streaming responses. Only set this when you set `stream: true`.

include\_obfuscation: optional boolean

When true, stream obfuscation will be enabled. Stream obfuscation adds random characters to an `obfuscation` field on streaming delta events to normalize payload sizes as a mitigation to certain side-channel attacks. These obfuscation fields are included by default, but add a small amount of overhead to the data stream. You can set `include_obfuscation` to false to optimize for bandwidth if you trust the network links between your application and the OpenAI API.

temperature: optional number

What sampling temperature to use, between 0 and 2. Higher values like 0.8 will make the output more random, while lower values like 0.2 will make it more focused and deterministic. We generally recommend altering this or `top_p` but not both.

minimum0

maximum2

text: optional [ResponseTextConfig](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_text_config%20%3E%20\(schema\)) { format, verbosity }

Configuration options for a text response from the model. Can be plain text or structured JSON data. Learn more:

- [Text inputs and outputs](https://developers.openai.com/docs/guides/text)
- [Structured Outputs](https://developers.openai.com/docs/guides/structured-outputs)

tool\_choice: optional [ToolChoiceOptions](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20tool_choice_options%20%3E%20\(schema\)) or [ToolChoiceAllowed](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20tool_choice_allowed%20%3E%20\(schema\)) { mode, tools, type } or [ToolChoiceTypes](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20tool_choice_types%20%3E%20\(schema\)) { type } or 5 more

How the model should select which tool (or tools) to use when generating a response. See the `tools` parameter to see how to specify which tools the model can call.

One of the following:

ToolChoiceOptions = "none" or "auto" or "required"

Controls which (if any) tool is called by the model.

`none` means the model will not call any tool and instead generates a message.

`auto` means the model can pick between generating a message or calling one or more tools.

`required` means the model must call one or more tools.

ToolChoiceAllowed object { mode, tools, type }

Constrains the tools available to the model to a pre-defined set.

mode: "auto" or "required"

Constrains the tools available to the model to a pre-defined set.

`auto` allows the model to pick from among the allowed tools and generate a message.

`required` requires the model to call one or more of the allowed tools.

One of the following:

"auto"

"required"

A list of tool definitions that the model should be allowed to call.

For the Responses API, the list of tool definitions might look like:

```json
[
  { "type": "function", "name": "get_weather" },
  { "type": "mcp", "server_label": "deepwiki" },
  { "type": "image_generation" }
]
```

type: "allowed\_tools"

Allowed tool configuration type. Always `allowed_tools`.

ToolChoiceTypes object { type }

Indicates that the model should use a built-in tool to generate a response. [Learn more about built-in tools](https://developers.openai.com/docs/guides/tools).

type: "file\_search" or "web\_search\_preview" or "computer" or 5 more

The type of hosted tool the model should to use. Learn more about [built-in tools](https://developers.openai.com/docs/guides/tools).

Allowed values are:

- `file_search`
- `web_search_preview`
- `computer`
- `computer_use_preview`
- `computer_use`
- `code_interpreter`
- `image_generation`

ToolChoiceFunction object { name, type }

Use this option to force the model to call a specific function.

name: string

The name of the function to call.

type: "function"

For function calling, the type is always `function`.

ToolChoiceMcp object { server\_label, type, name }

Use this option to force the model to call a specific tool on a remote MCP server.

server\_label: string

The label of the MCP server to use.

type: "mcp"

For MCP tools, the type is always `mcp`.

name: optional string

The name of the tool to call on the server.

ToolChoiceCustom object { name, type }

Use this option to force the model to call a specific custom tool.

name: string

The name of the custom tool to call.

type: "custom"

For custom tool calling, the type is always `custom`.

ToolChoiceApplyPatch object { type }

Forces the model to call the apply\_patch tool when executing a tool call.

type: "apply\_patch"

The tool to call. Always `apply_patch`.

ToolChoiceShell object { type }

Forces the model to call the shell tool when a tool call is required.

type: "shell"

The tool to call. Always `shell`.

tools: optional array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

An array of tools the model may call while generating a response. You can specify which tool to use by setting the `tool_choice` parameter.

We support the following categories of tools:

- **Built-in tools**: Tools that are provided by OpenAI that extend the model’s capabilities, like [web search](https://developers.openai.com/docs/guides/tools-web-search) or [file search](https://developers.openai.com/docs/guides/tools-file-search). Learn more about [built-in tools](https://developers.openai.com/docs/guides/tools).
- **MCP Tools**: Integrations with third-party systems via custom MCP servers or predefined connectors such as Google Drive and SharePoint. Learn more about [MCP Tools](https://developers.openai.com/docs/guides/tools-connectors-mcp).
- **Function calls (custom tools)**: Functions that are defined by you, enabling the model to call your own code with strongly typed arguments and outputs. Learn more about [function calling](https://developers.openai.com/docs/guides/function-calling). You can also use custom tools to call your own code.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

top\_logprobs: optional number

An integer between 0 and 20 specifying the maximum number of most likely tokens to return at each token position, each with an associated log probability. In some cases, the number of returned tokens may be fewer than requested.

minimum0

maximum20

top\_p: optional number

An alternative to sampling with temperature, called nucleus sampling, where the model considers the results of the tokens with top\_p probability mass. So 0.1 means only the tokens comprising the top 10% probability mass are considered.

We generally recommend altering this or `temperature` but not both.

minimum0

maximum1

truncation: optional "auto" or "disabled"

The truncation strategy to use for the model response.

- `auto`: If the input to this Response exceeds the model’s context window size, the model will truncate the response to fit the context window by dropping items from the beginning of the conversation.
- `disabled` (default): If the input size will exceed the context window size for a model, the request will fail with a 400 error.

One of the following:

"auto"

"disabled"

Deprecateduser: optional string

This field is being replaced by `safety_identifier` and `prompt_cache_key`. Use `prompt_cache_key` instead to maintain caching optimizations. A stable identifier for your end-users. Used to boost cache hit rates by better bucketing similar requests and to help OpenAI detect and prevent abuse. [Learn more](https://developers.openai.com/docs/guides/safety-best-practices#safety-identifiers).

ResponsesServerEvent = [ResponseAudioDeltaEvent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_audio_delta_event%20%3E%20\(schema\)) { delta, sequence\_number, type } or [ResponseAudioDoneEvent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_audio_done_event%20%3E%20\(schema\)) { sequence\_number, type } or [ResponseAudioTranscriptDeltaEvent](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_audio_transcript_delta_event%20%3E%20\(schema\)) { delta, sequence\_number, type } or 50 more

One of the following:

ResponseAudioDeltaEvent object { delta, sequence\_number, type }

Emitted when there is a partial audio response.

delta: string

A chunk of Base64 encoded response audio bytes.

sequence\_number: number

A sequence number for this chunk of the stream response.

type: "response.audio.delta"

The type of the event. Always `response.audio.delta`.

ResponseAudioDoneEvent object { sequence\_number, type }

Emitted when the audio response is complete.

sequence\_number: number

The sequence number of the delta.

type: "response.audio.done"

The type of the event. Always `response.audio.done`.

ResponseAudioTranscriptDeltaEvent object { delta, sequence\_number, type }

Emitted when there is a partial transcript of audio.

delta: string

The partial transcript of the audio response.

sequence\_number: number

The sequence number of this event.

type: "response.audio.transcript.delta"

The type of the event. Always `response.audio.transcript.delta`.

ResponseAudioTranscriptDoneEvent object { sequence\_number, type }

Emitted when the full audio transcript is completed.

sequence\_number: number

The sequence number of this event.

type: "response.audio.transcript.done"

The type of the event. Always `response.audio.transcript.done`.

ResponseCodeInterpreterCallCodeDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when a partial code snippet is streamed by the code interpreter.

delta: string

The partial code snippet being streamed by the code interpreter.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code is being streamed.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call\_code.delta"

The type of the event. Always `response.code_interpreter_call_code.delta`.

ResponseCodeInterpreterCallCodeDoneEvent object { code, item\_id, output\_index, 2 more }

Emitted when the code snippet is finalized by the code interpreter.

code: string

The final code snippet output by the code interpreter.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code is finalized.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call\_code.done"

The type of the event. Always `response.code_interpreter_call_code.done`.

ResponseCodeInterpreterCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the code interpreter call is completed.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter call is completed.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.completed"

The type of the event. Always `response.code_interpreter_call.completed`.

ResponseCodeInterpreterCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when a code interpreter call is in progress.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter call is in progress.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.in\_progress"

The type of the event. Always `response.code_interpreter_call.in_progress`.

ResponseCodeInterpreterCallInterpretingEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the code interpreter is actively interpreting the code snippet.

item\_id: string

The unique identifier of the code interpreter tool call item.

output\_index: number

The index of the output item in the response for which the code interpreter is interpreting code.

sequence\_number: number

The sequence number of this event, used to order streaming events.

type: "response.code\_interpreter\_call.interpreting"

The type of the event. Always `response.code_interpreter_call.interpreting`.

ResponseCompletedEvent object { response, sequence\_number, type }

Emitted when the model response is complete.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

Properties of the completed response.

sequence\_number: number

The sequence number for this event.

type: "response.completed"

The type of the event. Always `response.completed`.

ResponseContentPartAddedEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a new content part is added.

content\_index: number

The index of the content part that was added.

item\_id: string

The ID of the output item that the content part was added to.

output\_index: number

The index of the output item that the content part was added to.

part: [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type } or object { text, type }

The content part that was added.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

sequence\_number: number

The sequence number of this event.

type: "response.content\_part.added"

The type of the event. Always `response.content_part.added`.

ResponseContentPartDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a content part is done.

content\_index: number

The index of the content part that is done.

item\_id: string

The ID of the output item that the content part was added to.

output\_index: number

The index of the output item that the content part was added to.

part: [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type } or object { text, type }

The content part that is done.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

ReasoningText object { text, type }

Reasoning text from the model.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

sequence\_number: number

The sequence number of this event.

type: "response.content\_part.done"

The type of the event. Always `response.content_part.done`.

ResponseCreatedEvent object { response, sequence\_number, type }

An event that is emitted when a response is created.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that was created.

sequence\_number: number

The sequence number for this event.

type: "response.created"

The type of the event. Always `response.created`.

ResponseErrorEvent object { code, message, param, 2 more }

Emitted when an error occurs.

code: string

The error code.

message: string

The error message.

param: string

The error parameter.

sequence\_number: number

The sequence number of this event.

type: "error"

The type of the event. Always `error`.

ResponseFunctionCallArgumentsDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when there is a partial function-call arguments delta.

delta: string

The function-call arguments delta that is added.

item\_id: string

The ID of the output item that the function-call arguments delta is added to.

output\_index: number

The index of the output item that the function-call arguments delta is added to.

sequence\_number: number

The sequence number of this event.

type: "response.function\_call\_arguments.delta"

The type of the event. Always `response.function_call_arguments.delta`.

ResponseFunctionCallArgumentsDoneEvent object { arguments, item\_id, name, 3 more }

Emitted when function-call arguments are finalized.

arguments: string

The function-call arguments.

item\_id: string

The ID of the item.

name: string

The name of the function that was called.

output\_index: number

The index of the output item.

sequence\_number: number

The sequence number of this event.

type: "response.function\_call\_arguments.done"

ResponseInProgressEvent object { response, sequence\_number, type }

Emitted when the response is in progress.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that is in progress.

sequence\_number: number

The sequence number of this event.

type: "response.in\_progress"

The type of the event. Always `response.in_progress`.

ResponseFailedEvent object { response, sequence\_number, type }

An event that is emitted when a response fails.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that failed.

sequence\_number: number

The sequence number of this event.

type: "response.failed"

The type of the event. Always `response.failed`.

ResponseIncompleteEvent object { response, sequence\_number, type }

An event that is emitted when a response finishes as incomplete.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The response that was incomplete.

sequence\_number: number

The sequence number of this event.

type: "response.incomplete"

The type of the event. Always `response.incomplete`.

ResponseOutputItemAddedEvent object { item, output\_index, sequence\_number, type }

Emitted when a new output item is added.

item: [ResponseOutputItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_item%20%3E%20\(schema\))

The output item that was added.

output\_index: number

The index of the output item that was added.

sequence\_number: number

The sequence number of this event.

type: "response.output\_item.added"

The type of the event. Always `response.output_item.added`.

ResponseOutputItemDoneEvent object { item, output\_index, sequence\_number, type }

Emitted when an output item is marked done.

item: [ResponseOutputItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_item%20%3E%20\(schema\))

The output item that was marked done.

output\_index: number

The index of the output item that was marked done.

sequence\_number: number

The sequence number of this event.

type: "response.output\_item.done"

The type of the event. Always `response.output_item.done`.

ResponseReasoningSummaryPartAddedEvent object { item\_id, output\_index, part, 3 more }

Emitted when a new reasoning summary part is added.

item\_id: string

The ID of the item this summary part is associated with.

output\_index: number

The index of the output item this summary part is associated with.

part: object { text, type }

The summary part that was added.

text: string

The text of the summary part.

type: "summary\_text"

The type of the summary part. Always `summary_text`.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_part.added"

The type of the event. Always `response.reasoning_summary_part.added`.

ResponseReasoningSummaryPartDoneEvent object { item\_id, output\_index, part, 3 more }

Emitted when a reasoning summary part is completed.

item\_id: string

The ID of the item this summary part is associated with.

output\_index: number

The index of the output item this summary part is associated with.

part: object { text, type }

The completed summary part.

text: string

The text of the summary part.

type: "summary\_text"

The type of the summary part. Always `summary_text`.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_part.done"

The type of the event. Always `response.reasoning_summary_part.done`.

ResponseReasoningSummaryTextDeltaEvent object { delta, item\_id, output\_index, 3 more }

Emitted when a delta is added to a reasoning summary text.

delta: string

The text delta that was added to the summary.

item\_id: string

The ID of the item this summary text delta is associated with.

output\_index: number

The index of the output item this summary text delta is associated with.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

type: "response.reasoning\_summary\_text.delta"

The type of the event. Always `response.reasoning_summary_text.delta`.

ResponseReasoningSummaryTextDoneEvent object { item\_id, output\_index, sequence\_number, 3 more }

Emitted when a reasoning summary text is completed.

item\_id: string

The ID of the item this summary text is associated with.

output\_index: number

The index of the output item this summary text is associated with.

sequence\_number: number

The sequence number of this event.

summary\_index: number

The index of the summary part within the reasoning summary.

text: string

The full text of the completed reasoning summary.

type: "response.reasoning\_summary\_text.done"

The type of the event. Always `response.reasoning_summary_text.done`.

ResponseReasoningTextDeltaEvent object { content\_index, delta, item\_id, 3 more }

Emitted when a delta is added to a reasoning text.

content\_index: number

The index of the reasoning content part this delta is associated with.

delta: string

The text delta that was added to the reasoning content.

item\_id: string

The ID of the item this reasoning text delta is associated with.

output\_index: number

The index of the output item this reasoning text delta is associated with.

sequence\_number: number

The sequence number of this event.

type: "response.reasoning\_text.delta"

The type of the event. Always `response.reasoning_text.delta`.

ResponseReasoningTextDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when a reasoning text is completed.

content\_index: number

The index of the reasoning content part.

item\_id: string

The ID of the item this reasoning text is associated with.

output\_index: number

The index of the output item this reasoning text is associated with.

sequence\_number: number

The sequence number of this event.

text: string

The full text of the completed reasoning content.

type: "response.reasoning\_text.done"

The type of the event. Always `response.reasoning_text.done`.

ResponseRefusalDeltaEvent object { content\_index, delta, item\_id, 3 more }

Emitted when there is a partial refusal text.

content\_index: number

The index of the content part that the refusal text is added to.

delta: string

The refusal text that is added.

item\_id: string

The ID of the output item that the refusal text is added to.

output\_index: number

The index of the output item that the refusal text is added to.

sequence\_number: number

The sequence number of this event.

type: "response.refusal.delta"

The type of the event. Always `response.refusal.delta`.

ResponseRefusalDoneEvent object { content\_index, item\_id, output\_index, 3 more }

Emitted when refusal text is finalized.

content\_index: number

The index of the content part that the refusal text is finalized.

item\_id: string

The ID of the output item that the refusal text is finalized.

output\_index: number

The index of the output item that the refusal text is finalized.

refusal: string

The refusal text that is finalized.

sequence\_number: number

The sequence number of this event.

type: "response.refusal.done"

The type of the event. Always `response.refusal.done`.

ResponseTextDeltaEvent object { content\_index, delta, item\_id, 4 more }

Emitted when there is an additional text delta.

content\_index: number

The index of the content part that the text delta was added to.

delta: string

The text delta that was added.

item\_id: string

The ID of the output item that the text delta was added to.

logprobs: array of object { token, logprob, top\_logprobs }

The log probabilities of the tokens in the delta.

token: string

A possible text token.

logprob: number

The log probability of this token.

top\_logprobs: optional array of object { token, logprob }

The log probabilities of up to 20 of the most likely tokens.

token: optional string

A possible text token.

logprob: optional number

The log probability of this token.

output\_index: number

The index of the output item that the text delta was added to.

sequence\_number: number

The sequence number for this event.

type: "response.output\_text.delta"

The type of the event. Always `response.output_text.delta`.

ResponseTextDoneEvent object { content\_index, item\_id, logprobs, 4 more }

Emitted when text content is finalized.

content\_index: number

The index of the content part that the text content is finalized.

item\_id: string

The ID of the output item that the text content is finalized.

logprobs: array of object { token, logprob, top\_logprobs }

The log probabilities of the tokens in the delta.

token: string

A possible text token.

logprob: number

The log probability of this token.

top\_logprobs: optional array of object { token, logprob }

The log probabilities of up to 20 of the most likely tokens.

token: optional string

A possible text token.

logprob: optional number

The log probability of this token.

output\_index: number

The index of the output item that the text content is finalized.

sequence\_number: number

The sequence number for this event.

text: string

The text content that is finalized.

type: "response.output\_text.done"

The type of the event. Always `response.output_text.done`.

ResponseImageGenCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call has completed and the final image is available.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.image\_generation\_call.completed"

The type of the event. Always ‘response.image\_generation\_call.completed’.

ResponseImageGenCallGeneratingEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call is actively generating an image (intermediate state).

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.generating"

The type of the event. Always ‘response.image\_generation\_call.generating’.

ResponseImageGenCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an image generation tool call is in progress.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.in\_progress"

The type of the event. Always ‘response.image\_generation\_call.in\_progress’.

ResponseImageGenCallPartialImageEvent object { item\_id, output\_index, partial\_image\_b64, 3 more }

Emitted when a partial image is available during image generation streaming.

item\_id: string

The unique identifier of the image generation item being processed.

output\_index: number

The index of the output item in the response’s output array.

partial\_image\_b64: string

Base64-encoded partial image data, suitable for rendering as an image.

partial\_image\_index: number

0-based index for the partial image (backend is 1-based, but this is 0-based for the user).

sequence\_number: number

The sequence number of the image generation item being processed.

type: "response.image\_generation\_call.partial\_image"

The type of the event. Always ‘response.image\_generation\_call.partial\_image’.

ResponseMcpCallArgumentsDeltaEvent object { delta, item\_id, output\_index, 2 more }

Emitted when there is a delta (partial update) to the arguments of an MCP tool call.

delta: string

A JSON string containing the partial update to the arguments for the MCP tool call.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call\_arguments.delta"

The type of the event. Always ‘response.mcp\_call\_arguments.delta’.

ResponseMcpCallArgumentsDoneEvent object { arguments, item\_id, output\_index, 2 more }

Emitted when the arguments for an MCP tool call are finalized.

arguments: string

A JSON string containing the finalized arguments for the MCP tool call.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call\_arguments.done"

The type of the event. Always ‘response.mcp\_call\_arguments.done’.

ResponseMcpCallCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call has completed successfully.

item\_id: string

The ID of the MCP tool call item that completed.

output\_index: number

The index of the output item that completed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.completed"

The type of the event. Always ‘response.mcp\_call.completed’.

ResponseMcpCallFailedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call has failed.

item\_id: string

The ID of the MCP tool call item that failed.

output\_index: number

The index of the output item that failed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.failed"

The type of the event. Always ‘response.mcp\_call.failed’.

ResponseMcpCallInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when an MCP tool call is in progress.

item\_id: string

The unique identifier of the MCP tool call item being processed.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_call.in\_progress"

The type of the event. Always ‘response.mcp\_call.in\_progress’.

ResponseMcpListToolsCompletedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the list of available MCP tools has been successfully retrieved.

item\_id: string

The ID of the MCP tool call item that produced this output.

output\_index: number

The index of the output item that was processed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.completed"

The type of the event. Always ‘response.mcp\_list\_tools.completed’.

ResponseMcpListToolsFailedEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the attempt to list available MCP tools has failed.

item\_id: string

The ID of the MCP tool call item that failed.

output\_index: number

The index of the output item that failed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.failed"

The type of the event. Always ‘response.mcp\_list\_tools.failed’.

ResponseMcpListToolsInProgressEvent object { item\_id, output\_index, sequence\_number, type }

Emitted when the system is in the process of retrieving the list of available MCP tools.

item\_id: string

The ID of the MCP tool call item that is being processed.

output\_index: number

The index of the output item that is being processed.

sequence\_number: number

The sequence number of this event.

type: "response.mcp\_list\_tools.in\_progress"

The type of the event. Always ‘response.mcp\_list\_tools.in\_progress’.

ResponseOutputTextAnnotationAddedEvent object { annotation, annotation\_index, content\_index, 4 more }

Emitted when an annotation is added to output text content.

annotation: unknown

The annotation object being added. (See annotation schema for details.)

annotation\_index: number

The index of the annotation within the content part.

content\_index: number

The index of the content part within the output item.

item\_id: string

The unique identifier of the item to which the annotation is being added.

output\_index: number

The index of the output item in the response’s output array.

sequence\_number: number

The sequence number of this event.

type: "response.output\_text.annotation.added"

The type of the event. Always ‘response.output\_text.annotation.added’.

ResponseQueuedEvent object { response, sequence\_number, type }

Emitted when a response is queued and waiting to be processed.

response: [Response](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response%20%3E%20\(schema\)) { id, created\_at, error, 30 more }

The full response object that is queued.

sequence\_number: number

The sequence number for this event.

type: "response.queued"

The type of the event. Always ‘response.queued’.

ResponseCustomToolCallInputDeltaEvent object { delta, item\_id, output\_index, 2 more }

Event representing a delta (partial update) to the input of a custom tool call.

delta: string

The incremental input data (delta) for the custom tool call.

item\_id: string

Unique identifier for the API item associated with this event.

output\_index: number

The index of the output this delta applies to.

sequence\_number: number

The sequence number of this event.

type: "response.custom\_tool\_call\_input.delta"

The event type identifier.

ResponseCustomToolCallInputDoneEvent object { input, item\_id, output\_index, 2 more }

Event indicating that input for a custom tool call is complete.

input: string

The complete input data for the custom tool call.

item\_id: string

Unique identifier for the API item associated with this event.

output\_index: number

The index of the output this event applies to.

sequence\_number: number

The sequence number of this event.

type: "response.custom\_tool\_call\_input.done"

The event type identifier.

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

ToolChoiceAllowed object { mode, tools, type }

Constrains the tools available to the model to a pre-defined set.

mode: "auto" or "required"

Constrains the tools available to the model to a pre-defined set.

`auto` allows the model to pick from among the allowed tools and generate a message.

`required` requires the model to call one or more of the allowed tools.

One of the following:

"auto"

"required"

A list of tool definitions that the model should be allowed to call.

For the Responses API, the list of tool definitions might look like:

```json
[
  { "type": "function", "name": "get_weather" },
  { "type": "mcp", "server_label": "deepwiki" },
  { "type": "image_generation" }
]
```

type: "allowed\_tools"

Allowed tool configuration type. Always `allowed_tools`.

ToolChoiceApplyPatch object { type }

Forces the model to call the apply\_patch tool when executing a tool call.

type: "apply\_patch"

The tool to call. Always `apply_patch`.

ToolChoiceCustom object { name, type }

Use this option to force the model to call a specific custom tool.

name: string

The name of the custom tool to call.

type: "custom"

For custom tool calling, the type is always `custom`.

ToolChoiceFunction object { name, type }

Use this option to force the model to call a specific function.

name: string

The name of the function to call.

type: "function"

For function calling, the type is always `function`.

ToolChoiceMcp object { server\_label, type, name }

Use this option to force the model to call a specific tool on a remote MCP server.

server\_label: string

The label of the MCP server to use.

type: "mcp"

For MCP tools, the type is always `mcp`.

name: optional string

The name of the tool to call on the server.

ToolChoiceOptions = "none" or "auto" or "required"

Controls which (if any) tool is called by the model.

`none` means the model will not call any tool and instead generates a message.

`auto` means the model can pick between generating a message or calling one or more tools.

`required` means the model must call one or more tools.

ToolChoiceShell object { type }

Forces the model to call the shell tool when a tool call is required.

type: "shell"

The tool to call. Always `shell`.

ToolChoiceTypes object { type }

Indicates that the model should use a built-in tool to generate a response. [Learn more about built-in tools](https://developers.openai.com/docs/guides/tools).

type: "file\_search" or "web\_search\_preview" or "computer" or 5 more

The type of hosted tool the model should to use. Learn more about [built-in tools](https://developers.openai.com/docs/guides/tools).

Allowed values are:

- `file_search`
- `web_search_preview`
- `computer`
- `computer_use_preview`
- `computer_use`
- `code_interpreter`
- `image_generation`

#### ResponsesInput Items

##### ModelsExpand Collapse

ResponseItemList object { data, first\_id, has\_more, 2 more }

A list of Response items.

data: array of [ResponseInputMessageItem](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_item%20%3E%20\(schema\)) { id, content, role, 2 more } or [ResponseOutputMessage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_message%20%3E%20\(schema\)) { id, content, role, 3 more } or object { id, queries, status, 2 more } or 23 more

A list of items used to generate this response.

One of the following:

ResponseInputMessageItem object { id, content, role, 2 more }

id: string

The unique ID of the message input.

content: [ResponseInputMessageContentList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_message_content_list%20%3E%20\(schema\)) {,, }

A list of one or many input items to the model, containing different content types.

role: "user" or "system" or "developer"

The role of the message input. One of `user`, `system`, or `developer`.

type: "message"

The type of the message input. Always set to `message`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

ResponseOutputMessage object { id, content, role, 3 more }

An output message from the model.

id: string

The unique ID of the output message.

content: array of [ResponseOutputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_text%20%3E%20\(schema\)) { annotations, logprobs, text, type } or [ResponseOutputRefusal](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_output_refusal%20%3E%20\(schema\)) { refusal, type }

The content of the output message.

One of the following:

ResponseOutputText object { annotations, logprobs, text, type }

A text output from the model.

annotations: array of object { file\_id, filename, index, type } or object { end\_index, start\_index, title, 2 more } or object { container\_id, end\_index, file\_id, 3 more } or object { file\_id, index, type }

The annotations of the text output.

One of the following:

FileCitation object { file\_id, filename, index, type }

A citation to a file.

file\_id: string

The ID of the file.

filename: string

The filename of the file cited.

index: number

The index of the file in the list of files.

type: "file\_citation"

The type of the file citation. Always `file_citation`.

URLCitation object { end\_index, start\_index, title, 2 more }

A citation for a web resource used to generate a model response.

end\_index: number

The index of the last character of the URL citation in the message.

start\_index: number

The index of the first character of the URL citation in the message.

title: string

The title of the web resource.

type: "url\_citation"

The type of the URL citation. Always `url_citation`.

url: string

The URL of the web resource.

formaturi

ContainerFileCitation object { container\_id, end\_index, file\_id, 3 more }

A citation for a container file used to generate a model response.

container\_id: string

The ID of the container file.

end\_index: number

The index of the last character of the container file citation in the message.

file\_id: string

The ID of the file.

filename: string

The filename of the container file cited.

start\_index: number

The index of the first character of the container file citation in the message.

type: "container\_file\_citation"

The type of the container file citation. Always `container_file_citation`.

FilePath object { file\_id, index, type }

A path to a file.

file\_id: string

The ID of the file.

index: number

The index of the file in the list of files.

type: "file\_path"

The type of the file path. Always `file_path`.

logprobs: array of object { token, bytes, logprob, top\_logprobs }

token: string

bytes: array of number

logprob: number

top\_logprobs: array of object { token, bytes, logprob }

token: string

bytes: array of number

logprob: number

text: string

The text output from the model.

type: "output\_text"

The type of the output text. Always `output_text`.

ResponseOutputRefusal object { refusal, type }

A refusal from the model.

refusal: string

The refusal explanation from the model.

type: "refusal"

The type of the refusal. Always `refusal`.

role: "assistant"

The role of the output message. Always `assistant`.

status: "in\_progress" or "completed" or "incomplete"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "message"

The type of the output message. Always `message`.

phase: optional "commentary" or "final\_answer"

Labels an `assistant` message as intermediate commentary (`commentary`) or the final answer (`final_answer`). For models like `gpt-5.3-codex` and beyond, when sending follow-up requests, preserve and resend phase on all assistant messages — dropping it can degrade performance. Not used for user messages.

One of the following:

"commentary"

"final\_answer"

FileSearchCall object { id, queries, status, 2 more }

The results of a file search tool call. See the [file search guide](https://developers.openai.com/docs/guides/tools-file-search) for more information.

id: string

The unique ID of the file search tool call.

queries: array of string

The queries used to search for files.

status: "in\_progress" or "searching" or "completed" or 2 more

The status of the file search tool call. One of `in_progress`, `searching`, `incomplete` or `failed`,

type: "file\_search\_call"

The type of the file search tool call. Always `file_search_call`.

results: optional array of object { attributes, file\_id, filename, 2 more }

The results of the file search tool call.

attributes: optional map\[string or number or boolean\]

Set of 16 key-value pairs that can be attached to an object. This can be useful for storing additional information about the object in a structured format, and querying for objects via API or the dashboard. Keys are strings with a maximum length of 64 characters. Values are strings with a maximum length of 512 characters, booleans, or numbers.

file\_id: optional string

The unique ID of the file.

filename: optional string

The name of the file.

score: optional number

The relevance score of the file - a value between 0 and 1.

formatfloat

text: optional string

The text that was retrieved from the file.

ComputerCall object { id, call\_id, pending\_safety\_checks, 4 more }

A tool call to a computer use tool. See the [computer use guide](https://developers.openai.com/docs/guides/tools-computer-use) for more information.

id: string

The unique ID of the computer call.

call\_id: string

An identifier used when responding to the tool call with output.

pending\_safety\_checks: array of object { id, code, message }

The pending safety checks for the computer call.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "computer\_call"

The type of the computer call. Always `computer_call`.

action: optional [ComputerAction](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action%20%3E%20\(schema\))

A click action.

actions: optional [ComputerActionList](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20computer_action_list%20%3E%20\(schema\)) { Click, DoubleClick, Drag, 6 more }

Flattened batched actions for `computer_use`. Each action includes an `type` discriminator and action-specific fields.

ComputerCallOutput object { id, call\_id, output, 4 more }

id: string

The unique ID of the computer call tool output.

call\_id: string

The ID of the computer tool call that produced the output.

output: [ResponseComputerToolCallOutputScreenshot](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_computer_tool_call_output_screenshot%20%3E%20\(schema\)) { type, file\_id, image\_url }

A computer screenshot image used with the computer use tool.

status: "completed" or "incomplete" or "failed" or "in\_progress"

The status of the message input. One of `in_progress`, `completed`, or `incomplete`. Populated when input items are returned via API.

type: "computer\_call\_output"

The type of the computer tool call output. Always `computer_call_output`.

acknowledged\_safety\_checks: optional array of object { id, code, message }

The safety checks reported by the API that have been acknowledged by the developer.

id: string

The ID of the pending safety check.

code: optional string

The type of the pending safety check.

message: optional string

Details about the pending safety check.

created\_by: optional string

The identifier of the actor that created the item.

WebSearchCall object { id, action, status, type }

The results of a web search tool call. See the [web search guide](https://developers.openai.com/docs/guides/tools-web-search) for more information.

id: string

The unique ID of the web search tool call.

action: object { query, type, queries, sources } or object { type, url } or object { pattern, type, url }

An object describing the specific action taken in this web search call. Includes details on how the model used the web (search, open\_page, find\_in\_page).

One of the following:

Search object { query, type, queries, sources }

Action type “search” - Performs a web search query.

query: string

\[DEPRECATED\] The search query.

type: "search"

The action type.

queries: optional array of string

The search queries.

sources: optional array of object { type, url }

The sources used in the search.

type: "url"

The type of source. Always `url`.

url: string

The URL of the source.

formaturi

OpenPage object { type, url }

Action type “open\_page” - Opens a specific URL from search results.

type: "open\_page"

The action type.

url: optional string

The URL opened by the model.

formaturi

FindInPage object { pattern, type, url }

Action type “find\_in\_page”: Searches for a pattern within a loaded page.

pattern: string

The pattern or text to search for within the page.

type: "find\_in\_page"

The action type.

url: string

The URL of the page searched for the pattern.

formaturi

status: "in\_progress" or "searching" or "completed" or "failed"

The status of the web search tool call.

type: "web\_search\_call"

The type of the web search tool call. Always `web_search_call`.

FunctionCall object { id, arguments, call\_id, 5 more }

id: string

The unique ID of the function tool call.

arguments: string

A JSON string of the arguments to pass to the function.

call\_id: string

The unique ID of the function tool call generated by the model.

name: string

The name of the function to run.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "function\_call"

The type of the function tool call. Always `function_call`.

created\_by: optional string

The identifier of the actor that created the item.

namespace: optional string

The namespace of the function to run.

FunctionCallOutput object { id, call\_id, output, 3 more }

id: string

The unique ID of the function call tool output.

call\_id: string

The unique ID of the function tool call generated by the model.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the function call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the function call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the function call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "function\_call\_output"

The type of the function tool call output. Always `function_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

ToolSearchCall object { id, arguments, call\_id, 4 more }

id: string

The unique ID of the tool search call item.

arguments: unknown

Arguments used for the tool search call.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search call item that was recorded.

type: "tool\_search\_call"

The type of the item. Always `tool_search_call`.

created\_by: optional string

The identifier of the actor that created the item.

ToolSearchOutput object { id, call\_id, execution, 4 more }

id: string

The unique ID of the tool search output item.

call\_id: string

The unique ID of the tool search call generated by the model.

execution: "server" or "client"

Whether tool search was executed by the server or by the client.

One of the following:

"server"

"client"

status: "in\_progress" or "completed" or "incomplete"

The status of the tool search output item that was recorded.

tools: array of object { name, parameters, strict, 3 more } or object { type, vector\_store\_ids, filters, 2 more } or object { type } or 12 more

The loaded tool definitions returned by tool search.

One of the following:

Function object { name, parameters, strict, 3 more }

Defines a function in your own code the model can choose to call. Learn more about [function calling](https://platform.openai.com/docs/guides/function-calling).

name: string

The name of the function to call.

parameters: map\[unknown\]

A JSON schema object describing the parameters of the function.

strict: boolean

Whether to enforce strict parameter validation. Default `true`.

type: "function"

The type of the function tool. Always `function`.

description: optional string

A description of the function. Used by the model to determine whether or not to call the function.

FileSearch object { type, vector\_store\_ids, filters, 2 more }

A tool that searches for relevant content from uploaded files. Learn more about the [file search tool](https://platform.openai.com/docs/guides/tools-file-search).

type: "file\_search"

The type of the file search tool. Always `file_search`.

vector\_store\_ids: array of string

The IDs of the vector stores to search.

filters: optional [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or [CompoundFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20compound_filter%20%3E%20\(schema\)) { filters, type }

A filter to apply.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

CompoundFilter object { filters, type }

Combine multiple filters using `and` or `or`.

filters: array of [ComparisonFilter](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20comparison_filter%20%3E%20\(schema\)) { key, type, value } or unknown

Array of filters to combine. Items can be `ComparisonFilter` or `CompoundFilter`.

One of the following:

ComparisonFilter object { key, type, value }

A filter used to compare a specified attribute key to a given value using a defined comparison operation.

type: "eq" or "ne" or "gt" or 5 more

Specifies the comparison operator: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`.

- `eq`: equals
- `ne`: not equal
- `gt`: greater than
- `gte`: greater than or equal
- `lt`: less than
- `lte`: less than or equal
- `in`: in
- `nin`: not in

value: string or number or boolean or array of string or number

The value to compare against the attribute key; supports string, number, or boolean types.

unknown

One of the following:

"and"

"or"

max\_num\_results: optional number

The maximum number of results to return. This number should be between 1 and 50 inclusive.

ranking\_options: optional object { hybrid\_search, ranker, score\_threshold }

Ranking options for search.

ranker: optional "auto" or "default-2024-11-15"

The ranker to use for the file search.

One of the following:

"auto"

"default-2024-11-15"

score\_threshold: optional number

The score threshold for the file search, a number between 0 and 1. Numbers closer to 1 will attempt to return only the most relevant results, but may return fewer results.

Computer object { type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

type: "computer"

The type of the computer tool. Always `computer`.

ComputerUsePreview object { display\_height, display\_width, environment, type }

A tool that controls a virtual computer. Learn more about the [computer tool](https://platform.openai.com/docs/guides/tools-computer-use).

display\_height: number

The height of the computer display.

display\_width: number

The width of the computer display.

environment: "windows" or "mac" or "linux" or 2 more

The type of computer environment to control.

type: "computer\_use\_preview"

The type of the computer use tool. Always `computer_use_preview`.

WebSearch object { type, filters, search\_context\_size, user\_location }

Search the Internet for sources related to the prompt. Learn more about the [web search tool](https://developers.openai.com/docs/guides/tools-web-search).

type: "web\_search" or "web\_search\_2025\_08\_26"

The type of the web search tool. One of `web_search` or `web_search_2025_08_26`.

One of the following:

"web\_search"

"web\_search\_2025\_08\_26"

filters: optional object { allowed\_domains }

Filters for the search.

allowed\_domains: optional array of string

Allowed domains for the search. If not provided, all domains are allowed. Subdomains of the provided domains are allowed as well.

Example: `["pubmed.ncbi.nlm.nih.gov"]`

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { city, country, region, 2 more }

The approximate location of the user.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

type: optional "approximate"

The type of location approximation. Always `approximate`.

Mcp object { server\_label, type, allowed\_tools, 7 more }

Give the model access to additional tools via remote Model Context Protocol (MCP) servers. [Learn more about MCP](https://developers.openai.com/docs/guides/tools-remote-mcp).

server\_label: string

A label for this MCP server, used to identify it in tool calls.

type: "mcp"

The type of the MCP tool. Always `mcp`.

allowed\_tools: optional array of string or object { read\_only, tool\_names }

List of allowed tool names or a filter object.

One of the following:

McpAllowedTools = array of string

A string array of allowed tool names

McpToolFilter object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

authorization: optional string

An OAuth access token that can be used with a remote MCP server, either with a custom MCP server URL or a service connector. Your application must handle the OAuth authorization flow and provide the token here.

connector\_id: optional "connector\_dropbox" or "connector\_gmail" or "connector\_googlecalendar" or 5 more

Identifier for service connectors, like those available in ChatGPT. One of `server_url` or `connector_id` must be provided. Learn more about service connectors [here](https://developers.openai.com/docs/guides/tools-remote-mcp#connectors).

Currently supported `connector_id` values are:

- Dropbox: `connector_dropbox`
- Gmail: `connector_gmail`
- Google Calendar: `connector_googlecalendar`
- Google Drive: `connector_googledrive`
- Microsoft Teams: `connector_microsoftteams`
- Outlook Calendar: `connector_outlookcalendar`
- Outlook Email: `connector_outlookemail`
- SharePoint: `connector_sharepoint`

headers: optional map\[string\]

Optional HTTP headers to send to the MCP server. Use for authentication or other purposes.

require\_approval: optional object { always, never } or "always" or "never"

Specify which of the MCP server’s tools require approval.

One of the following:

McpToolApprovalFilter object { always, never }

Specify which of the MCP server’s tools require approval. Can be `always`, `never`, or a filter object associated with tools that require approval.

always: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

never: optional object { read\_only, tool\_names }

A filter object to specify which tools are allowed.

read\_only: optional boolean

Indicates whether or not a tool modifies data or is read-only. If an MCP server is [annotated with `readOnlyHint`](https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations-readonlyhint), it will match this filter.

tool\_names: optional array of string

List of allowed tool names.

McpToolApprovalSetting = "always" or "never"

Specify a single approval policy for all tools. One of `always` or `never`. When set to `always`, all tools will require approval. When set to `never`, all tools will not require approval.

One of the following:

"always"

"never"

server\_description: optional string

Optional description of the MCP server, used to provide more context.

server\_url: optional string

The URL for the MCP server. One of `server_url` or `connector_id` must be provided.

formaturi

CodeInterpreter object { container, type }

A tool that runs Python code to help generate a response to a prompt.

container: string or object { type, file\_ids, memory\_limit, network\_policy }

The code interpreter container. Can be a container ID or an object that specifies uploaded file IDs to make available to your code, along with an optional `memory_limit` setting.

One of the following:

string

The container ID.

CodeInterpreterToolAuto object { type, file\_ids, memory\_limit, network\_policy }

Configuration for a code interpreter container. Optionally specify the IDs of the files to run the code on.

type: "auto"

Always `auto`.

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the code interpreter container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

type: "code\_interpreter"

The type of the code interpreter tool. Always `code_interpreter`.

ImageGeneration object { type, action, background, 9 more }

A tool that generates images using the GPT image models.

type: "image\_generation"

The type of the image generation tool. Always `image_generation`.

action: optional "generate" or "edit" or "auto"

Whether to generate a new image or edit an existing image. Default: `auto`.

background: optional "transparent" or "opaque" or "auto"

Background type for the generated image. One of `transparent`, `opaque`, or `auto`. Default: `auto`.

input\_fidelity: optional "high" or "low"

Control how much effort the model will exert to match the style and features, especially facial features, of input images. This parameter is only supported for `gpt-image-1` and `gpt-image-1.5` and later models, unsupported for `gpt-image-1-mini`. Supports `high` and `low`. Defaults to `low`.

One of the following:

"high"

"low"

input\_image\_mask: optional object { file\_id, image\_url }

Optional mask for inpainting. Contains `image_url` (string, optional) and `file_id` (string, optional).

file\_id: optional string

File ID for the mask image.

image\_url: optional string

Base64-encoded mask image.

model: optional string or "gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

One of the following:

string

"gpt-image-1" or "gpt-image-1-mini" or "gpt-image-1.5"

The image generation model to use. Default: `gpt-image-1`.

moderation: optional "auto" or "low"

Moderation level for the generated image. Default: `auto`.

One of the following:

"auto"

"low"

output\_compression: optional number

Compression level for the output image. Default: 100.

minimum0

maximum100

output\_format: optional "png" or "webp" or "jpeg"

The output format of the generated image. One of `png`, `webp`, or `jpeg`. Default: `png`.

partial\_images: optional number

Number of partial images to generate in streaming mode, from 0 (default value) to 3.

minimum0

maximum3

quality: optional "low" or "medium" or "high" or "auto"

The quality of the generated image. One of `low`, `medium`, `high`, or `auto`. Default: `auto`.

size: optional "1024x1024" or "1024x1536" or "1536x1024" or "auto"

The size of the generated image. One of `1024x1024`, `1024x1536`, `1536x1024`, or `auto`. Default: `auto`.

LocalShell object { type }

A tool that allows the model to execute shell commands in a local environment.

type: "local\_shell"

The type of the local shell tool. Always `local_shell`.

Shell object { type, environment }

A tool that allows the model to execute shell commands.

type: "shell"

The type of the shell tool. Always `shell`.

environment: optional [ContainerAuto](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_auto%20%3E%20\(schema\)) { type, file\_ids, memory\_limit, 2 more } or [LocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_environment%20%3E%20\(schema\)) { type, skills } or [ContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_reference%20%3E%20\(schema\)) { container\_id, type }

One of the following:

ContainerAuto object { type, file\_ids, memory\_limit, 2 more }

type: "container\_auto"

Automatically creates a container for this request

file\_ids: optional array of string

An optional list of uploaded files to make available to your code.

memory\_limit: optional "1g" or "4g" or "16g" or "64g"

The memory limit for the container.

network\_policy: optional [ContainerNetworkPolicyDisabled](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_disabled%20%3E%20\(schema\)) { type } or [ContainerNetworkPolicyAllowlist](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_allowlist%20%3E%20\(schema\)) { allowed\_domains, type, domain\_secrets }

Network access policy for the container.

One of the following:

ContainerNetworkPolicyDisabled object { type }

type: "disabled"

Disable outbound network access. Always `disabled`.

ContainerNetworkPolicyAllowlist object { allowed\_domains, type, domain\_secrets }

allowed\_domains: array of string

A list of allowed domains when type is `allowlist`.

type: "allowlist"

Allow outbound network access only to specified domains. Always `allowlist`.

domain\_secrets: optional array of [ContainerNetworkPolicyDomainSecret](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20container_network_policy_domain_secret%20%3E%20\(schema\)) { domain, name, value }

Optional domain-scoped secrets for allowlisted domains.

domain: string

The domain associated with the secret.

minLength1

name: string

The name of the secret to inject for the domain.

minLength1

value: string

The secret value to inject for the domain.

maxLength10485760

minLength1

skills: optional array of [SkillReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20skill_reference%20%3E%20\(schema\)) { skill\_id, type, version } or [InlineSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill%20%3E%20\(schema\)) { description, name, source, type }

An optional list of skills referenced by id or inline data.

One of the following:

SkillReference object { skill\_id, type, version }

skill\_id: string

The ID of the referenced skill.

maxLength64

minLength1

type: "skill\_reference"

References a skill created with the /v1/skills endpoint.

version: optional string

Optional skill version. Use a positive integer or ‘latest’. Omit for default.

InlineSkill object { description, name, source, type }

description: string

The description of the skill.

name: string

The name of the skill.

source: [InlineSkillSource](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20inline_skill_source%20%3E%20\(schema\)) { data, media\_type, type }

Inline skill payload

type: "inline"

Defines an inline skill for this request.

LocalEnvironment object { type, skills }

type: "local"

Use a local computer environment.

skills: optional array of [LocalSkill](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20local_skill%20%3E%20\(schema\)) { description, name, path }

An optional list of skills.

description: string

The description of the skill.

name: string

The name of the skill.

path: string

The path to the directory containing the skill.

ContainerReference object { container\_id, type }

container\_id: string

The ID of the referenced container.

type: "container\_reference"

References a container created with the /v1/containers endpoint

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

Namespace object { description, name, tools, type }

Groups function/custom tools under a shared namespace.

description: string

A description of the namespace shown to the model.

minLength1

name: string

The namespace name used in tool calls (for example, `crm`).

minLength1

tools: array of object { name, type, defer\_loading, 3 more } or object { name, type, defer\_loading, 2 more }

The function/custom tools available inside this namespace.

One of the following:

Function object { name, type, defer\_loading, 3 more }

name: string

maxLength128

minLength1

type: "function"

description: optional string

parameters: optional unknown

strict: optional boolean

Custom object { name, type, defer\_loading, 2 more }

A custom tool that processes input using a specified format. Learn more about [custom tools](https://developers.openai.com/docs/guides/function-calling#custom-tools)

name: string

The name of the custom tool, used to identify it in tool calls.

type: "custom"

The type of the custom tool. Always `custom`.

description: optional string

Optional description of the custom tool, used to provide more context.

format: optional [CustomToolInputFormat](https://developers.openai.com/api/reference/resources/$shared#\(resource\)%20%24shared%20%3E%20\(model\)%20custom_tool_input_format%20%3E%20\(schema\))

The input format for the custom tool. Default is unconstrained text.

type: "namespace"

The type of the tool. Always `namespace`.

ToolSearch object { type, description, execution, parameters }

Hosted or BYOT tool search configuration for deferred tools.

type: "tool\_search"

The type of the tool. Always `tool_search`.

description: optional string

Description shown to the model for a client-executed tool search tool.

execution: optional "server" or "client"

Whether tool search is executed by the server or by the client.

One of the following:

"server"

"client"

parameters: optional unknown

Parameter schema for a client-executed tool search tool.

WebSearchPreview object { type, search\_content\_types, search\_context\_size, user\_location }

This tool searches the web for relevant results to use in a response. Learn more about the [web search tool](https://platform.openai.com/docs/guides/tools-web-search).

type: "web\_search\_preview" or "web\_search\_preview\_2025\_03\_11"

The type of the web search tool. One of `web_search_preview` or `web_search_preview_2025_03_11`.

One of the following:

"web\_search\_preview"

"web\_search\_preview\_2025\_03\_11"

search\_content\_types: optional array of "text" or "image"

One of the following:

"text"

"image"

search\_context\_size: optional "low" or "medium" or "high"

High level guidance for the amount of context window space to use for the search. One of `low`, `medium`, or `high`. `medium` is the default.

user\_location: optional object { type, city, country, 2 more }

The user’s location.

type: "approximate"

The type of location approximation. Always `approximate`.

city: optional string

Free text input for the city of the user, e.g. `San Francisco`.

country: optional string

The two-letter [ISO country code](https://en.wikipedia.org/wiki/ISO_3166-1) of the user, e.g. `US`.

region: optional string

Free text input for the region of the user, e.g. `California`.

timezone: optional string

The [IANA timezone](https://timeapi.io/documentation/iana-timezones) of the user, e.g. `America/Los_Angeles`.

ApplyPatch object { type }

Allows the assistant to create, delete, or update files using unified diffs.

type: "apply\_patch"

The type of the tool. Always `apply_patch`.

type: "tool\_search\_output"

The type of the item. Always `tool_search_output`.

created\_by: optional string

The identifier of the actor that created the item.

Reasoning object { id, summary, type, 3 more }

A description of the chain of thought used by a reasoning model while generating a response. Be sure to include these items in your `input` to the Responses API for subsequent turns of a conversation if you are manually [managing context](https://developers.openai.com/docs/guides/conversation-state).

id: string

The unique identifier of the reasoning content.

summary: array of [SummaryTextContent](https://developers.openai.com/api/reference/resources/conversations#\(resource\)%20conversations%20%3E%20\(model\)%20summary_text_content%20%3E%20\(schema\)) { text, type }

Reasoning summary content.

text: string

A summary of the reasoning output from the model so far.

type: "summary\_text"

The type of the object. Always `summary_text`.

type: "reasoning"

The type of the object. Always `reasoning`.

content: optional array of object { text, type }

Reasoning text content.

text: string

The reasoning text from the model.

type: "reasoning\_text"

The type of the reasoning text. Always `reasoning_text`.

encrypted\_content: optional string

The encrypted content of the reasoning item - populated when a response is generated with `reasoning.encrypted_content` in the `include` parameter.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

Compaction object { id, encrypted\_content, type, created\_by }

id: string

The unique ID of the compaction item.

encrypted\_content: string

The encrypted content that was produced by compaction.

type: "compaction"

The type of the item. Always `compaction`.

created\_by: optional string

The identifier of the actor that created the item.

ImageGenerationCall object { id, result, status, type }

An image generation request made by the model.

id: string

The unique ID of the image generation call.

result: string

The generated image encoded in base64.

status: "in\_progress" or "completed" or "generating" or "failed"

The status of the image generation call.

type: "image\_generation\_call"

The type of the image generation call. Always `image_generation_call`.

CodeInterpreterCall object { id, code, container\_id, 3 more }

A tool call to run code.

id: string

The unique ID of the code interpreter tool call.

code: string

The code to run, or null if not available.

container\_id: string

The ID of the container used to run the code.

outputs: array of object { logs, type } or object { type, url }

The outputs generated by the code interpreter, such as logs or images. Can be null if no outputs are available.

One of the following:

Logs object { logs, type }

The logs output from the code interpreter.

logs: string

The logs output from the code interpreter.

type: "logs"

The type of the output. Always `logs`.

Image object { type, url }

The image output from the code interpreter.

type: "image"

The type of the output. Always `image`.

url: string

The URL of the image output from the code interpreter.

formaturi

status: "in\_progress" or "completed" or "incomplete" or 2 more

The status of the code interpreter tool call. Valid values are `in_progress`, `completed`, `incomplete`, `interpreting`, and `failed`.

type: "code\_interpreter\_call"

The type of the code interpreter tool call. Always `code_interpreter_call`.

LocalShellCall object { id, action, call\_id, 2 more }

A tool call to run a command on the local shell.

id: string

The unique ID of the local shell call.

action: object { command, env, type, 3 more }

Execute a shell command on the server.

command: array of string

The command to run.

env: map\[string\]

Environment variables to set for the command.

type: "exec"

The type of the local shell action. Always `exec`.

timeout\_ms: optional number

Optional timeout in milliseconds for the command.

user: optional string

Optional user to run the command as.

working\_directory: optional string

Optional working directory to run the command in.

call\_id: string

The unique ID of the local shell tool call generated by the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the local shell call.

type: "local\_shell\_call"

The type of the local shell call. Always `local_shell_call`.

LocalShellCallOutput object { id, output, type, status }

The output of a local shell tool call.

id: string

The unique ID of the local shell tool call generated by the model.

output: string

A JSON string of the output of the local shell tool call.

type: "local\_shell\_call\_output"

The type of the local shell tool call output. Always `local_shell_call_output`.

status: optional "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`.

ShellCall object { id, action, call\_id, 4 more }

A tool call that executes one or more shell commands in a managed environment.

id: string

The unique ID of the shell tool call. Populated when this item is returned via API.

action: object { commands, max\_output\_length, timeout\_ms }

The shell commands and limits that describe how to run the tool call.

commands: array of string

max\_output\_length: number

Optional maximum number of characters to return from each command.

timeout\_ms: number

Optional timeout in milliseconds for the commands.

call\_id: string

The unique ID of the shell tool call generated by the model.

environment: [ResponseLocalEnvironment](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_local_environment%20%3E%20\(schema\)) { type } or [ResponseContainerReference](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_container_reference%20%3E%20\(schema\)) { container\_id, type }

Represents the use of a local environment to perform shell actions.

One of the following:

ResponseLocalEnvironment object { type }

Represents the use of a local environment to perform shell actions.

type: "local"

The environment type. Always `local`.

ResponseContainerReference object { container\_id, type }

Represents a container created with /v1/containers.

container\_id: string

type: "container\_reference"

The environment type. Always `container_reference`.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call"

The type of the item. Always `shell_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ShellCallOutput object { id, call\_id, max\_output\_length, 4 more }

The output of a shell tool call that was emitted.

id: string

The unique ID of the shell call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the shell tool call generated by the model.

max\_output\_length: number

The maximum length of the shell command output. This is generated by the model and should be passed back with the raw output.

output: array of object { outcome, stderr, stdout, created\_by }

An array of shell call output contents

outcome: object { type } or object { exit\_code, type }

Represents either an exit outcome (with an exit code) or a timeout outcome for a shell call output chunk.

One of the following:

Timeout object { type }

Indicates that the shell call exceeded its configured time limit.

type: "timeout"

The outcome type. Always `timeout`.

Exit object { exit\_code, type }

Indicates that the shell commands finished and returned an exit code.

exit\_code: number

Exit code from the shell process.

type: "exit"

The outcome type. Always `exit`.

stderr: string

The standard error output that was captured.

stdout: string

The standard output that was captured.

created\_by: optional string

The identifier of the actor that created the item.

status: "in\_progress" or "completed" or "incomplete"

The status of the shell call output. One of `in_progress`, `completed`, or `incomplete`.

type: "shell\_call\_output"

The type of the shell call output. Always `shell_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

ApplyPatchCall object { id, call\_id, operation, 3 more }

A tool call that applies file diffs by creating, deleting, or updating files.

id: string

The unique ID of the apply patch tool call. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

operation: object { diff, path, type } or object { path, type } or object { diff, path, type }

One of the create\_file, delete\_file, or update\_file operations applied via apply\_patch.

One of the following:

CreateFile object { diff, path, type }

Instruction describing how to create a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to create.

type: "create\_file"

Create a new file with the provided diff.

DeleteFile object { path, type }

Instruction describing how to delete a file via the apply\_patch tool.

path: string

Path of the file to delete.

type: "delete\_file"

Delete the specified file.

UpdateFile object { diff, path, type }

Instruction describing how to update a file via the apply\_patch tool.

diff: string

Diff to apply.

path: string

Path of the file to update.

type: "update\_file"

Update an existing file with the provided diff.

status: "in\_progress" or "completed"

The status of the apply patch tool call. One of `in_progress` or `completed`.

One of the following:

"in\_progress"

"completed"

type: "apply\_patch\_call"

The type of the item. Always `apply_patch_call`.

created\_by: optional string

The ID of the entity that created this tool call.

ApplyPatchCallOutput object { id, call\_id, status, 3 more }

The output emitted by an apply patch tool call.

id: string

The unique ID of the apply patch tool call output. Populated when this item is returned via API.

call\_id: string

The unique ID of the apply patch tool call generated by the model.

status: "completed" or "failed"

The status of the apply patch tool call output. One of `completed` or `failed`.

One of the following:

"completed"

"failed"

type: "apply\_patch\_call\_output"

The type of the item. Always `apply_patch_call_output`.

created\_by: optional string

The ID of the entity that created this tool call output.

output: optional string

Optional textual output returned by the apply patch tool.

McpListTools object { id, server\_label, tools, 2 more }

A list of tools available on an MCP server.

id: string

The unique ID of the list.

server\_label: string

The label of the MCP server.

tools: array of object { input\_schema, name, annotations, description }

The tools available on the server.

input\_schema: unknown

The JSON schema describing the tool’s input.

name: string

The name of the tool.

annotations: optional unknown

Additional annotations about the tool.

description: optional string

The description of the tool.

type: "mcp\_list\_tools"

The type of the item. Always `mcp_list_tools`.

error: optional string

Error message if the server could not list tools.

McpApprovalRequest object { id, arguments, name, 2 more }

A request for human approval of a tool invocation.

id: string

The unique ID of the approval request.

arguments: string

A JSON string of arguments for the tool.

name: string

The name of the tool to run.

server\_label: string

The label of the MCP server making the request.

type: "mcp\_approval\_request"

The type of the item. Always `mcp_approval_request`.

McpApprovalResponse object { id, approval\_request\_id, approve, 2 more }

A response to an MCP approval request.

id: string

The unique ID of the approval response

approval\_request\_id: string

The ID of the approval request being answered.

approve: boolean

Whether the request was approved.

type: "mcp\_approval\_response"

The type of the item. Always `mcp_approval_response`.

reason: optional string

Optional reason for the decision.

McpCall object { id, arguments, name, 6 more }

An invocation of a tool on an MCP server.

id: string

The unique ID of the tool call.

arguments: string

A JSON string of the arguments passed to the tool.

name: string

The name of the tool that was run.

server\_label: string

The label of the MCP server running the tool.

type: "mcp\_call"

The type of the item. Always `mcp_call`.

approval\_request\_id: optional string

Unique identifier for the MCP tool call approval request. Include this value in a subsequent `mcp_approval_response` input to approve or reject the corresponding tool call.

error: optional string

The error from the tool call, if any.

output: optional string

The output from the tool call.

status: optional "in\_progress" or "completed" or "incomplete" or 2 more

The status of the tool call. One of `in_progress`, `completed`, `incomplete`, `calling`, or `failed`.

CustomToolCall object { id, call\_id, input, 5 more }

id: string

The unique ID of the custom tool call item.

call\_id: string

An identifier used to map this custom tool call to a tool call output.

input: string

The input for the custom tool call generated by the model.

name: string

The name of the custom tool being called.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "custom\_tool\_call"

The type of the custom tool call. Always `custom_tool_call`.

created\_by: optional string

The identifier of the actor that created the item.

namespace: optional string

The namespace of the custom tool being called.

CustomToolCallOutput object { id, call\_id, output, 3 more }

id: string

The unique ID of the custom tool call output item.

call\_id: string

The call ID, used to map this custom tool call output to a custom tool call.

output: string or array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

The output from the custom tool call generated by your code. Can be a string or an list of output content.

One of the following:

StringOutput = string

A string of the output of the custom tool call.

OutputContentList = array of [ResponseInputText](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_text%20%3E%20\(schema\)) { text, type } or [ResponseInputImage](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_image%20%3E%20\(schema\)) { detail, type, file\_id, image\_url } or [ResponseInputFile](https://developers.openai.com/api/reference/resources/responses#\(resource\)%20responses%20%3E%20\(model\)%20response_input_file%20%3E%20\(schema\)) { type, detail, file\_data, 3 more }

Text, image, or file output of the custom tool call.

One of the following:

ResponseInputText object { text, type }

A text input to the model.

text: string

The text input to the model.

type: "input\_text"

The type of the input item. Always `input_text`.

ResponseInputImage object { detail, type, file\_id, image\_url }

An image input to the model. Learn about [image inputs](https://developers.openai.com/docs/guides/vision).

detail: "low" or "high" or "auto" or "original"

The detail level of the image to be sent to the model. One of `high`, `low`, `auto`, or `original`. Defaults to `auto`.

type: "input\_image"

The type of the input item. Always `input_image`.

file\_id: optional string

The ID of the file to be sent to the model.

image\_url: optional string

The URL of the image to be sent to the model. A fully qualified URL or base64 encoded image in a data URL.

formaturi

ResponseInputFile object { type, detail, file\_data, 3 more }

A file input to the model.

type: "input\_file"

The type of the input item. Always `input_file`.

detail: optional "low" or "high"

The detail level of the file to be sent to the model. Use `low` for the default rendering behavior, or `high` to render the file at higher quality. Defaults to `low`.

One of the following:

"low"

"high"

file\_data: optional string

The content of the file to be sent to the model.

file\_id: optional string

The ID of the file to be sent to the model.

file\_url: optional string

The URL of the file to be sent to the model.

formaturi

filename: optional string

The name of the file to be sent to the model.

status: "in\_progress" or "completed" or "incomplete"

The status of the item. One of `in_progress`, `completed`, or `incomplete`. Populated when items are returned via API.

type: "custom\_tool\_call\_output"

The type of the custom tool call output. Always `custom_tool_call_output`.

created\_by: optional string

The identifier of the actor that created the item.

first\_id: string

The ID of the first item in the list.

has\_more: boolean

Whether there are more items available.

last\_id: string

The ID of the last item in the list.

object: "list"

The type of object returned, must be `list`.
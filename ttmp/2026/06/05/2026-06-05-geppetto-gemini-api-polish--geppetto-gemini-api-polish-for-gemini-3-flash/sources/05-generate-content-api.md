---
Title: Gemini Generate Content API Reference
SourceURL: https://ai.google.dev/api/generate-content?hl=en
SourceTool: defuddle
FetchedAt: 2026-06-05T09:02:40-04:00
Ticket: 2026-06-05-geppetto-gemini-api-polish
Topics:
  - geppetto
  - providers
  - reasoning
  - streaming
  - tools
DocType: source
Status: active
Intent: long-term
Owners:
  - manuel
RelatedFiles: []
ExternalSources: []
Summary: Official Gemini or SDK reference captured for the Geppetto Gemini API polish ticket.
WhatFor: Use as source material for Gemini 3 Flash API compatibility, thinking, thought signatures, function calling, and SDK migration.
WhenToUse: Read before changing Geppetto's Gemini provider implementation.
---

The Gemini API supports content generation with images, audio, code, tools, and more. For details
on each of these features, read on and check out the task-focused sample code, or read the
comprehensive guides.

- [Text generation](https://ai.google.dev/gemini-api/docs/text-generation)
- [Vision](https://ai.google.dev/gemini-api/docs/vision)
- [Audio](https://ai.google.dev/gemini-api/docs/audio)
- [Embeddings](https://ai.google.dev/gemini-api/docs/embeddings)
- [Long context](https://ai.google.dev/gemini-api/docs/long-context)
- [Code execution](https://ai.google.dev/gemini-api/docs/code-execution)
- [JSON Mode](https://ai.google.dev/gemini-api/docs/json-mode)
- [Function calling](https://ai.google.dev/gemini-api/docs/function-calling)
- [System instructions](https://ai.google.dev/gemini-api/docs/system-instructions)

## Method: models.generateContent

Generates a model response given an input `GenerateContentRequest`. Refer to the [text generation
guide](https://ai.google.dev/gemini-api/docs/text-generation) for detailed usage information. Input
capabilities differ between models, including tuned models. Refer to the [model
guide](https://ai.google.dev/gemini-api/docs/models/gemini) and [tuning
guide](https://ai.google.dev/gemini-api/docs/model-tuning) for details.

### Endpoint

post `https://generativelanguage.googleapis.com/v1beta/{model=models/*}:generateContent`

### Path parameters

`model` `string`

Required. The name of the `Model` to use for generating the completion.

Format: `models/{model}`. It takes the form `models/{model}`.

### Request body

The request body contains data with the following structure:

Fields

`contents[]` ``object (`Content`)``

Required. The content of the current conversation with the model.

For single-turn queries, this is a single instance. For multi-turn queries like
[chat](https://ai.google.dev/gemini-api/docs/text-generation#chat), this is a repeated field that
contains the conversation history and the latest request.

`tools[]` ``object (`Tool`)``

Optional. A list of `Tools` the `Model` may use to generate the next response.

A `Tool` is a piece of code that enables the system to interact with external systems to perform an
action, or set of actions, outside of knowledge and scope of the `Model`. Supported `Tool` s are
`Function` and `codeExecution`. Refer to the [Function
calling](https://ai.google.dev/gemini-api/docs/function-calling) and the [Code
execution](https://ai.google.dev/gemini-api/docs/code-execution) guides to learn more.

`toolConfig` ``object (`ToolConfig`)``

Optional. Tool configuration for any `Tool` specified in the request. Refer to the [Function
calling guide](https://ai.google.dev/gemini-api/docs/function-calling#function_calling_mode) for a
usage example.

`safetySettings[]` ``object (`SafetySetting`)``

Optional. A list of unique `SafetySetting` instances for blocking unsafe content.

This will be enforced on the `GenerateContentRequest.contents` and
`GenerateContentResponse.candidates`. There should not be more than one setting for each
`SafetyCategory` type. The API will block any contents and responses that fail to meet the
thresholds set by these settings. This list overrides the default settings for each
`SafetyCategory` specified in the safetySettings. If there is no `SafetySetting` for a given
`SafetyCategory` provided in the list, the API will use the default safety setting for that
category. Harm categories HARM\_CATEGORY\_HATE\_SPEECH, HARM\_CATEGORY\_SEXUALLY\_EXPLICIT,
HARM\_CATEGORY\_DANGEROUS\_CONTENT, HARM\_CATEGORY\_HARASSMENT, HARM\_CATEGORY\_CIVIC\_INTEGRITY
are supported. Refer to the [guide](https://ai.google.dev/gemini-api/docs/safety-settings) for
detailed information on available safety settings. Also refer to the [Safety
guidance](https://ai.google.dev/gemini-api/docs/safety-guidance) to learn how to incorporate safety
considerations in your AI applications.

`systemInstruction` ``object (`Content`)``

Optional. Developer set [system
instruction(s)](https://ai.google.dev/gemini-api/docs/system-instructions). Currently, text only.

`generationConfig` ``object (`GenerationConfig`)``

Optional. Configuration options for model generation and outputs.

`cachedContent` `string`

Optional. The name of the content [cached](https://ai.google.dev/gemini-api/docs/caching) to use as
context to serve the prediction. Format: `cachedContents/{cachedContent}`

`serviceTier` ``enum (`ServiceTier`)``

Optional. The service tier of the request.

`store` `boolean`

Optional. Configures the logging behavior for a given request. If set, it takes precedence over the
project-level logging config.

### Example request

### Text

### Python

```python
from google import genai

client = genai.Client()
response = client.models.generate_content(
    model="gemini-3.5-flash", contents="Write a story about a magic backpack."
)
print(response.text)text_generation.py
```

### Node.js

```javascript
// Make sure to include the following import:
// import {GoogleGenAI} from '@google/genai';
const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY });

const response = await ai.models.generateContent({
  model: "gemini-3.5-flash",
  contents: "Write a story about a magic backpack.",
});
console.log(response.text);text_generation.js
```

### Go

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}
contents := []*genai.Content{
    genai.NewContentFromText("Write a story about a magic backpack.", genai.RoleUser),
}
response, err := client.Models.GenerateContent(ctx, "gemini-3.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}
printResponse(response)text_generation.go
```

### Shell

```shell
curl
"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GEMIN
I_API_KEY" \
    -H 'Content-Type: application/json' \
    -X POST \
    -d '{
      "contents": [{
        "parts":[{"text": "Write a story about a magic backpack."}]
        }]
       }' 2> /dev/nulltext_generation.sh
```

### Java

```java
Client client = new Client();

GenerateContentResponse response =
        client.models.generateContent(
                "gemini-3.5-flash",
                "Write a story about a magic backpack.",
                null);

System.out.println(response.text());TextGeneration.java
```

### Image

### Python

```python
from google import genai
import PIL.Image

client = genai.Client()
organ = PIL.Image.open(media / "organ.jpg")
response = client.models.generate_content(
    model="gemini-3.5-flash", contents=["Tell me about this instrument", organ]
)
print(response.text)text_generation.py
```

### Node.js

```javascript
// Make sure to include the following import:
// import {GoogleGenAI} from '@google/genai';
const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY });

const organ = await ai.files.upload({
  file: path.join(media, "organ.jpg"),
});

const response = await ai.models.generateContent({
  model: "gemini-3.5-flash",
  contents: [
    createUserContent([
      "Tell me about this instrument",
      createPartFromUri(organ.uri, organ.mimeType)
    ]),
  ],
});
console.log(response.text);text_generation.js
```

### Go

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}

file, err := client.Files.UploadFromPath(
    ctx,
    filepath.Join(getMedia(), "organ.jpg"),
    &genai.UploadFileConfig{
        MIMEType : "image/jpeg",
    },
)
if err != nil {
    log.Fatal(err)
}
parts := []*genai.Part{
    genai.NewPartFromText("Tell me about this instrument"),
    genai.NewPartFromURI(file.URI, file.MIMEType),
}
contents := []*genai.Content{
    genai.NewContentFromParts(parts, genai.RoleUser),
}

response, err := client.Models.GenerateContent(ctx, "gemini-3.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}
printResponse(response)text_generation.go
```

### Shell

```shell
# Use a temporary file to hold the base64 encoded image data
TEMP_B64=$(mktemp)
trap 'rm -f "$TEMP_B64"' EXIT
base64 $B64FLAGS $IMG_PATH > "$TEMP_B64"

# Use a temporary file to hold the JSON payload
TEMP_JSON=$(mktemp)
trap 'rm -f "$TEMP_JSON"' EXIT

cat > "$TEMP_JSON" << EOF
{
  "contents": [{
    "parts":[
      {"text": "Tell me about this instrument"},
      {
        "inline_data": {
          "mime_type":"image/jpeg",
          "data": "$(cat "$TEMP_B64")"
        }
      }
    ]
  }]
}
EOF

curl
"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GEMIN
I_API_KEY" \
    -H 'Content-Type: application/json' \
    -X POST \
    -d "@$TEMP_JSON" 2> /dev/nulltext_generation.sh
```

### Java

```java
Client client = new Client();

String path = media_path + "organ.jpg";
byte[] imageData = Files.readAllBytes(Paths.get(path));

Content content =
        Content.fromParts(
                Part.fromText("Tell me about this instrument."),
                Part.fromBytes(imageData, "image/jpeg"));

GenerateContentResponse response = client.models.generateContent("gemini-3.5-flash", content, null);

System.out.println(response.text());TextGeneration.java
```

### Audio

### Python

```python
from google import genai

client = genai.Client()
sample_audio = client.files.upload(file=media / "sample.mp3")
response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents=["Give me a summary of this audio file.", sample_audio],
)
print(response.text)text_generation.py
```

### Node.js

```javascript
// Make sure to include the following import:
// import {GoogleGenAI} from '@google/genai';
const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY });

const audio = await ai.files.upload({
  file: path.join(media, "sample.mp3"),
});

const response = await ai.models.generateContent({
  model: "gemini-3.5-flash",
  contents: [
    createUserContent([
      "Give me a summary of this audio file.",
      createPartFromUri(audio.uri, audio.mimeType),
    ]),
  ],
});
console.log(response.text);text_generation.js
```

### Go

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}

file, err := client.Files.UploadFromPath(
    ctx,
    filepath.Join(getMedia(), "sample.mp3"),
    &genai.UploadFileConfig{
        MIMEType : "audio/mpeg",
    },
)
if err != nil {
    log.Fatal(err)
}

parts := []*genai.Part{
    genai.NewPartFromText("Give me a summary of this audio file."),
    genai.NewPartFromURI(file.URI, file.MIMEType),
}

contents := []*genai.Content{
    genai.NewContentFromParts(parts, genai.RoleUser),
}

response, err := client.Models.GenerateContent(ctx, "gemini-3.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}
printResponse(response)text_generation.go
```

### Shell

```shell
# Use File API to upload audio data to API request.
MIME_TYPE=$(file -b --mime-type "${AUDIO_PATH}")
NUM_BYTES=$(wc -c < "${AUDIO_PATH}")
DISPLAY_NAME=AUDIO

tmp_header_file=upload-header.tmp

# Initial resumable request defining metadata.
# The upload url is in the response headers dump them to a file.
curl "${BASE_URL}/upload/v1beta/files?key=${GEMINI_API_KEY}" \
  -D upload-header.tmp \
  -H "X-Goog-Upload-Protocol: resumable" \
  -H "X-Goog-Upload-Command: start" \
  -H "X-Goog-Upload-Header-Content-Length: ${NUM_BYTES}" \
  -H "X-Goog-Upload-Header-Content-Type: ${MIME_TYPE}" \
  -H "Content-Type: application/json" \
  -d "{'file': {'display_name': '${DISPLAY_NAME}'}}" 2> /dev/null

upload_url=$(grep -i "x-goog-upload-url: " "${tmp_header_file}" | cut -d" " -f2 | tr -d "\r")
rm "${tmp_header_file}"

# Upload the actual bytes.
curl "${upload_url}" \
  -H "Content-Length: ${NUM_BYTES}" \
  -H "X-Goog-Upload-Offset: 0" \
  -H "X-Goog-Upload-Command: upload, finalize" \
  --data-binary "@${AUDIO_PATH}" 2> /dev/null > file_info.json

file_uri=$(jq ".file.uri" file_info.json)
echo file_uri=$file_uri

curl
"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GEMIN
I_API_KEY" \
    -H 'Content-Type: application/json' \
    -X POST \
    -d '{
      "contents": [{
        "parts":[
          {"text": "Please describe this file."},
          {"file_data":{"mime_type": "audio/mpeg", "file_uri": '$file_uri'}}]
        }]
       }' 2> /dev/null > response.json

cat response.json
echo

jq ".candidates[].content.parts[].text" response.jsontext_generation.sh
```

### Video

### Python

```python
from google import genai
import time

client = genai.Client()
# Video clip (CC BY 3.0) from https://peach.blender.org/download/
myfile = client.files.upload(file=media / "Big_Buck_Bunny.mp4")
print(f"{myfile=}")

# Poll until the video file is completely processed (state becomes ACTIVE).
while not myfile.state or myfile.state.name != "ACTIVE":
    print("Processing video...")
    print("File state:", myfile.state)
    time.sleep(5)
    myfile = client.files.get(name=myfile.name)

response = client.models.generate_content(
    model="gemini-3.5-flash", contents=[myfile, "Describe this video clip"]
)
print(f"{response.text=}")text_generation.py
```

### Node.js

```javascript
// Make sure to include the following import:
// import {GoogleGenAI} from '@google/genai';
const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY });

let video = await ai.files.upload({
  file: path.join(media, 'Big_Buck_Bunny.mp4'),
});

// Poll until the video file is completely processed (state becomes ACTIVE).
while (!video.state || video.state.toString() !== 'ACTIVE') {
  console.log('Processing video...');
  console.log('File state: ', video.state);
  await sleep(5000);
  video = await ai.files.get({name: video.name});
}

const response = await ai.models.generateContent({
  model: "gemini-3.5-flash",
  contents: [
    createUserContent([
      "Describe this video clip",
      createPartFromUri(video.uri, video.mimeType),
    ]),
  ],
});
console.log(response.text);text_generation.js
```

### Go

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}

file, err := client.Files.UploadFromPath(
    ctx,
    filepath.Join(getMedia(), "Big_Buck_Bunny.mp4"),
    &genai.UploadFileConfig{
        MIMEType : "video/mp4",
    },
)
if err != nil {
    log.Fatal(err)
}

// Poll until the video file is completely processed (state becomes ACTIVE).
for file.State == genai.FileStateUnspecified || file.State != genai.FileStateActive {
    fmt.Println("Processing video...")
    fmt.Println("File state:", file.State)
    time.Sleep(5 * time.Second)

    file, err = client.Files.Get(ctx, file.Name, nil)
    if err != nil {
        log.Fatal(err)
    }
}

parts := []*genai.Part{
    genai.NewPartFromText("Describe this video clip"),
    genai.NewPartFromURI(file.URI, file.MIMEType),
}

contents := []*genai.Content{
    genai.NewContentFromParts(parts, genai.RoleUser),
}

response, err := client.Models.GenerateContent(ctx, "gemini-3.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}
printResponse(response)text_generation.go
```

### Shell

```shell
# Use File API to upload audio data to API request.
MIME_TYPE=$(file -b --mime-type "${VIDEO_PATH}")
NUM_BYTES=$(wc -c < "${VIDEO_PATH}")
DISPLAY_NAME=VIDEO

# Initial resumable request defining metadata.
# The upload url is in the response headers dump them to a file.
curl "${BASE_URL}/upload/v1beta/files?key=${GEMINI_API_KEY}" \
  -D "${tmp_header_file}" \
  -H "X-Goog-Upload-Protocol: resumable" \
  -H "X-Goog-Upload-Command: start" \
  -H "X-Goog-Upload-Header-Content-Length: ${NUM_BYTES}" \
  -H "X-Goog-Upload-Header-Content-Type: ${MIME_TYPE}" \
  -H "Content-Type: application/json" \
  -d "{'file': {'display_name': '${DISPLAY_NAME}'}}" 2> /dev/null

upload_url=$(grep -i "x-goog-upload-url: " "${tmp_header_file}" | cut -d" " -f2 | tr -d "\r")
rm "${tmp_header_file}"

# Upload the actual bytes.
curl "${upload_url}" \
  -H "Content-Length: ${NUM_BYTES}" \
  -H "X-Goog-Upload-Offset: 0" \
  -H "X-Goog-Upload-Command: upload, finalize" \
  --data-binary "@${VIDEO_PATH}" 2> /dev/null > file_info.json

file_uri=$(jq ".file.uri" file_info.json)
echo file_uri=$file_uri

state=$(jq ".file.state" file_info.json)
echo state=$state

name=$(jq ".file.name" file_info.json)
echo name=$name

while [[ "($state)" = *"PROCESSING"* ]];
do
  echo "Processing video..."
  sleep 5
  # Get the file of interest to check state
  curl https://generativelanguage.googleapis.com/v1beta/files/$name > file_info.json
  state=$(jq ".file.state" file_info.json)
done

curl
"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GEMIN
I_API_KEY" \
    -H 'Content-Type: application/json' \
    -X POST \
    -d '{
      "contents": [{
        "parts":[
          {"text": "Transcribe the audio from this video, giving timestamps for salient events in
the video. Also provide visual descriptions."},
          {"file_data":{"mime_type": "video/mp4", "file_uri": '$file_uri'}}]
        }]
       }' 2> /dev/null > response.json

cat response.json
echo

jq ".candidates[].content.parts[].text" response.jsontext_generation.sh
```

### PDF

### Python

```python
from google import genai

client = genai.Client()
sample_pdf = client.files.upload(file=media / "test.pdf")
response = client.models.generate_content(
    model="gemini-3.5-flash",
    contents=["Give me a summary of this document:", sample_pdf],
)
print(f"{response.text=}")text_generation.py
```

### Go

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}

file, err := client.Files.UploadFromPath(
    ctx,
    filepath.Join(getMedia(), "test.pdf"),
    &genai.UploadFileConfig{
        MIMEType : "application/pdf",
    },
)
if err != nil {
    log.Fatal(err)
}

parts := []*genai.Part{
    genai.NewPartFromText("Give me a summary of this document:"),
    genai.NewPartFromURI(file.URI, file.MIMEType),
}

contents := []*genai.Content{
    genai.NewContentFromParts(parts, genai.RoleUser),
}

response, err := client.Models.GenerateContent(ctx, "gemini-3.5-flash", contents, nil)
if err != nil {
    log.Fatal(err)
}
printResponse(response)text_generation.go
```

### Shell

```shell
MIME_TYPE=$(file -b --mime-type "${PDF_PATH}")
NUM_BYTES=$(wc -c < "${PDF_PATH}")
DISPLAY_NAME=TEXT

echo $MIME_TYPE
tmp_header_file=upload-header.tmp

# Initial resumable request defining metadata.
# The upload url is in the response headers dump them to a file.
curl "${BASE_URL}/upload/v1beta/files?key=${GEMINI_API_KEY}" \
  -D upload-header.tmp \
  -H "X-Goog-Upload-Protocol: resumable" \
  -H "X-Goog-Upload-Command: start" \
  -H "X-Goog-Upload-Header-Content-Length: ${NUM_BYTES}" \
  -H "X-Goog-Upload-Header-Content-Type: ${MIME_TYPE}" \
  -H "Content-Type: application/json" \
  -d "{'file': {'display_name': '${DISPLAY_NAME}'}}" 2> /dev/null

upload_url=$(grep -i "x-goog-upload-url: " "${tmp_header_file}" | cut -d" " -f2 | tr -d "\r")
rm "${tmp_header_file}"

# Upload the actual bytes.
curl "${upload_url}" \
  -H "Content-Length: ${NUM_BYTES}" \
  -H "X-Goog-Upload-Offset: 0" \
  -H "X-Goog-Upload-Command: upload, finalize" \
  --data-binary "@${PDF_PATH}" 2> /dev/null > file_info.json

file_uri=$(jq ".file.uri" file_info.json)
echo file_uri=$file_uri

# Now generate content using that file
curl
"https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GEMIN
I_API_KEY" \
    -H 'Content-Type: application/json' \
    -X POST \
    -d '{
      "contents": [{
        "parts":[
          {"text": "Can you add a few more lines to this poem?"},
          {"file_data":{"mime_type": "application/pdf", "file_uri": '$file_uri'}}]
        }]
       }' 2> /dev/null > response.json

cat response.json
echo

jq ".candidates[].content.parts[].text" response.jsontext_generation.sh
```

### Chat

### Python

```python
from google import genai
from google.genai import types

client = genai.Client()
# Pass initial history using the "history" argument
chat = client.chats.create(
    model="gemini-3.5-flash",
    history=[
        types.Content(role="user", parts=[types.Part(text="Hello")]),
        types.Content(
            role="model",
            parts=[
                types.Part(
                    text="Great to meet you. What would you like to know?"
                )
            ],
        ),
    ],
)
response = chat.send_message(message="I have 2 dogs in my house.")
print(response.text)
response = chat.send_message(message="How many paws are in my house?")
print(response.text)chat.py
```

### Node.js

```javascript
// Make sure to include the following import:
// import {GoogleGenAI} from '@google/genai';
const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY });
const chat = ai.chats.create({
  model: "gemini-3.5-flash",
  history: [
    {
      role: "user",
      parts: [{ text: "Hello" }],
    },
    {
      role: "model",
      parts: [{ text: "Great to meet you. What would you like to know?" }],
    },
  ],
});

const response1 = await chat.sendMessage({
  message: "I have 2 dogs in my house.",
});
console.log("Chat response 1:", response1.text);

const response2 = await chat.sendMessage({
  message: "How many paws are in my house?",
});
console.log("Chat response 2:", response2.text);chat.js
```

### Go

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}

// Pass initial history using the History field.
history := []*genai.Content{
    genai.NewContentFromText("Hello", genai.RoleUser),
    genai.NewContentFromText("Great to meet you. What would you like to know?", genai.RoleModel),
}

chat, err := client.Chats.Create(ctx, "gemini-3.5-flash", nil, history)
if err != nil {
    log.Fatal(err)
}

firstResp, err := chat.SendMessage(ctx, genai.Part{Text: "I have 2 dogs in my house."})
if err != nil {
    log.Fatal(err)
}
fmt.Println(firstResp.Text())

secondResp, err := chat.SendMessage(ctx, genai.Part{Text: "How many paws are in my house?"})
if err != nil {
    log.Fatal(err)
}
fmt.Println(secondResp.Text())chat.go
```

### Shell

```shell
curl
https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=$GEMINI
_API_KEY \
    -H 'Content-Type: application/json' \
    -X POST \
    -d '{
      "contents": [
        {"role":"user",
         "parts":[{
           "text": "Hello"}]},
        {"role": "model",
         "parts":[{
           "text": "Great to meet you. What would you like to know?"}]},
        {"role":"user",
         "parts":[{
           "text": "I have two dogs in my house. How many paws are in my house?"}]},
      ]
    }' 2> /dev/null | grep "text"chat.sh
```

### Java

```java
Client client = new Client();

Content userContent = Content.fromParts(Part.fromText("Hello"));
Content modelContent =
        Content.builder()
                .role("model")
                .parts(
                        Collections.singletonList(
                                Part.fromText("Great to meet you. What would you like to know?")
                        )
                ).build();

Chat chat = client.chats.create(
        "gemini-3.5-flash",
        GenerateContentConfig.builder()
                .systemInstruction(userContent)
                .systemInstruction(modelContent)
                .build()
);

GenerateContentResponse response1 = chat.sendMessage("I have 2 dogs in my house.");
System.out.println(response1.text());

GenerateContentResponse response2 = chat.sendMessage("How many paws are in my house?");
System.out.println(response2.text());
ChatSession.java
```

### Cache

### Python

```python
from google import genai
from google.genai import types

client = genai.Client()
document = client.files.upload(file=media / "a11.txt")
model_name = "gemini-3.5-flash"

cache = client.caches.create(
    model=model_name,
    config=types.CreateCachedContentConfig(
        contents=[document],
        system_instruction="You are an expert analyzing transcripts.",
    ),
)
print(cache)

response = client.models.generate_content(
    model=model_name,
    contents="Please summarize this transcript",
    config=types.GenerateContentConfig(cached_content=cache.name),
)
print(response.text)cache.py
```

### Node.js

```javascript
// Make sure to include the following import:
// import {GoogleGenAI} from '@google/genai';
const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY });
const filePath = path.join(media, "a11.txt");
const document = await ai.files.upload({
  file: filePath,
  config: { mimeType: "text/plain" },
});
console.log("Uploaded file name:", document.name);
const modelName = "gemini-3.5-flash";

const contents = [
  createUserContent(createPartFromUri(document.uri, document.mimeType)),
];

const cache = await ai.caches.create({
  model: modelName,
  config: {
    contents: contents,
    systemInstruction: "You are an expert analyzing transcripts.",
  },
});
console.log("Cache created:", cache);

const response = await ai.models.generateContent({
  model: modelName,
  contents: "Please summarize this transcript",
  config: { cachedContent: cache.name },
});
console.log("Response text:", response.text);cache.js
```

### Go

```go
ctx := context.Background()
client, err := genai.NewClient(ctx, &genai.ClientConfig{
    APIKey:  os.Getenv("GEMINI_API_KEY"),
    Backend: genai.BackendGeminiAPI,
})
if err != nil {
    log.Fatal(err)
}

modelName := "gemini-3.5-flash"
document, err := client.Files.UploadFromPath(
    ctx,
    filepath.Join(getMedia(), "a11.txt"),
    &genai.UploadFileConfig{
        MIMEType : "text/plain",
    },
)
if err != nil {
    log.Fatal(err)
}
parts := []*genai.Part{
    genai.NewPartFromURI(document.URI, document.MIMEType),
}
contents := []*genai.Content{
    genai.NewContentFromParts(parts, genai.RoleUser),
}
cache, err := client.Caches.Create(ctx, modelName, &genai.CreateCachedContentConfig{
    Contents: contents,
    SystemInstruction: genai.NewContentFromText(
        "You are an expert analyzing transcripts.", genai.RoleUser,
    ),
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("Cache created:")
fmt.Println(cache)

// Use the cache for generating content.
response, err := client.Models.GenerateContent(
    ctx,
    modelName,
    genai.Text("Please summarize this transcript"),
    &genai.GenerateContentConfig{
        CachedContent: cache.Name,
    },
)
if err != nil {
    log.Fatal(err)
}
printResponse(response)cache.go
```

### Tuned Model

### Python

```python
# With Gemini 2 we're launching a new SDK. See the following doc for details.
# https://ai.google.dev/gemini-api/docs/migrateREADME.md
```

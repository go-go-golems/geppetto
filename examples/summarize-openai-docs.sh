#!/bin/bash

json_data=$(go run ./cmd/pinocchio openai ls-families --output json --fields name,description)

# Parse the JSON data using jq
echo "$json_data" | jq -c '.[]' | while read item; do
  name=$(echo "$item" | jq -r '.name')
  description=$(echo "$item" | jq -r '.description')

  # Use the extracted values as flags for the program
  echo $description | go run ./cmd/pinocchio prompts summarize-model-description \
      --instructions "Only one short sentence, no more than 20 words." \
      --name "$name" -
done
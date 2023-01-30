- **Max tokens** - {{ .Completion.MaxTokens }}
- **Training data cutoff date** - {{ .Completion.TrainingDataCutoffDate }}
- **Price per 1000 tokens** - {{ .Family.PricePer1kTokens }}

## Description

{{ .Completion.Description }}

## {{ .Completion.Family }}

### Key points

{{ range $keyPoints := .Family.KeyPoints }}
- {{ . }}
{{ end }}

### Good At

{{ range $goodAt := .Family.GoodAt }}
- {{ . }}
{{ end }}

### Description

{{ .Family.Description }}

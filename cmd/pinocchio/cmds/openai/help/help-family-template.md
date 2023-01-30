**Price per 1000 tokens** - {{ .PricePer1kTokens }}
 
## Key points

{{ range $keyPoints := .KeyPoints }}
- {{ . }}
{{ end }}

## Good At

{{ range $goodAt := .GoodAt }}
- {{ . }}
{{ end }}

## Description

{{ .Description }}
package parse

import (
    "strings"

    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/text"
)

// ExtractYAMLBlocks scans a markdown string and returns the contents of fenced YAML/YML code blocks.
// It preserves the inner code content without the enclosing fences.
func ExtractYAMLBlocks(markdownText string) ([]string, error) {
    var results []string
    source := []byte(markdownText)
    doc := goldmark.DefaultParser().Parse(text.NewReader(source))

    err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
        if !entering {
            return ast.WalkContinue, nil
        }
        if cb, ok := n.(*ast.FencedCodeBlock); ok {
            lang := strings.ToLower(string(cb.Language(source)))
            if lang == "yaml" || lang == "yml" {
                // Collect lines
                if cb.Lines().Len() > 0 {
                    start := cb.Lines().At(0).Start
                    stop := cb.Lines().At(cb.Lines().Len()-1).Stop
                    code := string(source[start:stop])
                    results = append(results, code)
                }
            }
        }
        return ast.WalkContinue, nil
    })
    if err != nil {
        return nil, err
    }
    return results, nil
}



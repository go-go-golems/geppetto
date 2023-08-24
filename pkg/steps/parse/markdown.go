package parse

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type ParseMarkdownStep struct{}

type Header struct {
	Text  string
	Level int
}

type CodeBlock struct {
	Code     string
	Language string
}

type Paragraph struct {
	Text string
}

type List struct {
	Entries []string
}

type QuotedText struct {
	Text string
}

type Link struct {
	Destination []byte
	Title       []byte
}

type Image struct {
	Destination []byte
	Title       []byte
}

func ExtractContentFromMarkdown(markdownText string) ([]interface{}, error) {
	var content []interface{}
	source := []byte(markdownText)

	document := goldmark.DefaultParser().Parse(
		text.NewReader(source),
	)

	err := ast.Walk(document, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch v := n.(type) {
			case *ast.Heading:
				content = append(content, Header{
					Text:  string(v.Text(source)),
					Level: v.Level,
				})
			case *ast.FencedCodeBlock:
				content = append(content, CodeBlock{
					Code:     string(source[v.Lines().At(0).Start:v.Lines().At(v.Lines().Len()-1).Stop]),
					Language: string(v.Language(source)),
				})
			case *ast.Paragraph:
				content = append(content, Paragraph{
					Text: string(v.Text(source)),
				})
			case *ast.List:
				var list List
				cur := v.FirstChild()
				for cur != nil {
					text_ := cur.Text(source)
					cur = cur.NextSibling()
					list.Entries = append(list.Entries, string(text_))
				}
				content = append(content, list)
			case *ast.Blockquote:
				content = append(content, QuotedText{
					Text: string(v.Text(source)),
				})
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}

	return content, nil
}

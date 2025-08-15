package turns

// TurnBuilder helps construct an initial Turn with ordered Blocks.
type TurnBuilder struct {
	blocks []Block
}

func NewTurnBuilder() *TurnBuilder {
	return &TurnBuilder{blocks: []Block{}}
}

func (tb *TurnBuilder) WithSystemPrompt(systemText string) *TurnBuilder {
	if systemText != "" {
		tb.blocks = append(tb.blocks, NewSystemTextBlock(systemText))
	}
	return tb
}

func (tb *TurnBuilder) WithUserPrompt(userText string) *TurnBuilder {
	if userText != "" {
		tb.blocks = append(tb.blocks, NewUserTextBlock(userText))
	}
	return tb
}

func (tb *TurnBuilder) Build() *Turn {
	t := &Turn{}
	if len(tb.blocks) > 0 {
		AppendBlocks(t, tb.blocks...)
	}
	return t
}

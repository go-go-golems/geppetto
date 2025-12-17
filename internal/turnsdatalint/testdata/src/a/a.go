package a

import "github.com/go-go-golems/geppetto/pkg/turns"

func okConst(t *turns.Turn) {
	_ = t.Data[turns.DataKeyToolRegistry]

	const Local turns.TurnDataKey = "local"
	_ = t.Data[Local]

	const Indirect = turns.DataKeyAgentMode
	_ = t.Data[Indirect]
}

func badConversion(t *turns.Turn) {
	_ = t.Data[turns.TurnDataKey("raw")] // want `Turn\.Data key must be a const of type`
}

func badVar(t *turns.Turn) {
	k := turns.DataKeyToolRegistry
	_ = t.Data[k] // want `Turn\.Data key must be a const of type`
}



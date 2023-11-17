package ui

import (
	"fmt"
	"github.com/muesli/reflow/wordwrap"
)

func wrapWords(text string, w int) string {
	w_ := wordwrap.NewWriter(w)
	_, err := fmt.Fprint(w_, text)
	if err != nil {
		panic(err)
	}
	_ = w_.Close()
	v := w_.String()
	return v
}

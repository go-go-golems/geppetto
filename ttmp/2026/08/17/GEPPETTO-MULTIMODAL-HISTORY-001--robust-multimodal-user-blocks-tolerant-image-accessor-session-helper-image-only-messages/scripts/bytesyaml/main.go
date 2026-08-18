// Experiment: how does serde.ToYAML encode []byte image content, and what comes back?
package main

import (
	"fmt"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/geppetto/pkg/turns/serde"
)

func main() {
	t := &turns.Turn{}
	turns.AppendBlock(t, turns.NewUserMultimodalBlock("x", []map[string]any{{"media_type": "image/jpeg", "content": []byte("JPG")}}))
	y, _ := serde.ToYAML(t, serde.Options{})
	fmt.Println(string(y))
	back, _ := serde.FromYAML(y)
	fmt.Printf("%T %#v\n", turns.BlockImages(back.Blocks[0])[0]["content"], turns.BlockImages(back.Blocks[0])[0]["content"])
}

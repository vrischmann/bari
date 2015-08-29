package bari_test

import (
	"fmt"
	"strings"

	"github.com/vrischmann/bari"
)

func ExampleParser_Parse_single() {
	const data = `{"foo": "bar"}`

	parser := bari.NewParser(strings.NewReader(data))
	ch := make(chan bari.Event)

	go func() {
		parser.Parse(ch)
		close(ch)
	}()

	for ev := range ch {
		fmt.Println(ev.Type, ev.Value)
	}
	// Output:
	// ObjectStartEvent <nil>
	// ObjectKeyEvent <nil>
	// StringEvent foo
	// ObjectValueEvent <nil>
	// StringEvent bar
	// ObjectEndEvent <nil>
}

func ExampleParser_Parse_multi() {
	const data = `{"foo": "bar"}{"bar": true}`

	parser := bari.NewParser(strings.NewReader(data))
	ch := make(chan bari.Event)

	go func() {
		parser.Parse(ch)
		close(ch)
	}()

	for ev := range ch {
		fmt.Println(ev.Type, ev.Value)
	}
	// Output:
	// ObjectStartEvent <nil>
	// ObjectKeyEvent <nil>
	// StringEvent foo
	// ObjectValueEvent <nil>
	// StringEvent bar
	// ObjectEndEvent <nil>
	// ObjectStartEvent <nil>
	// ObjectKeyEvent <nil>
	// StringEvent bar
	// ObjectValueEvent <nil>
	// BooleanEvent true
	// ObjectEndEvent <nil>
}

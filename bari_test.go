package bari_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vrischmann/bari"
)

func TestParse(t *testing.T) {
	const data = `{"foo": "bar"}`

	parser := bari.NewParser(strings.NewReader(data))
	ch := make(chan bari.Event)

	go func() {
		parser.Parse(ch)
		close(ch)
	}()

	for evt := range ch {
		fmt.Printf("%+v\n", evt)
	}
}

func TestParseNumber(t *testing.T) {
	const data = `{"foo": 10.0}`

	parser := bari.NewParser(strings.NewReader(data))
	ch := make(chan bari.Event)

	go func() {
		parser.Parse(ch)
		close(ch)
	}()

	for evt := range ch {
		fmt.Printf("%+v\n", evt)
	}
}

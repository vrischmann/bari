package bari_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrischmann/bari"
)

func ck(t testing.TB, evt bari.Event, typ bari.EventType, value interface{}, err error) {
	require.Equal(t, typ, evt.Type)
	require.Equal(t, value, evt.Value)
	require.Equal(t, err, evt.Error)
}

func TestParse(t *testing.T) {
	const data = `{"foo": "bar"}`

	parser := bari.NewParser(strings.NewReader(data))
	ch := make(chan bari.Event)

	go func() {
		parser.Parse(ch)
		close(ch)
	}()

	ck(t, <-ch, bari.ObjectStartEvent, nil, nil)
	ck(t, <-ch, bari.ObjectKeyEvent, nil, nil)
	ck(t, <-ch, bari.StringEvent, "foo", nil)
	ck(t, <-ch, bari.ObjectValueEvent, nil, nil)
	ck(t, <-ch, bari.StringEvent, "bar", nil)
}

func TestParseNumber(t *testing.T) {
	const data = `{"foo": 10.0}`

	parser := bari.NewParser(strings.NewReader(data))
	ch := make(chan bari.Event)

	go func() {
		parser.Parse(ch)
		close(ch)
	}()

	ck(t, <-ch, bari.ObjectStartEvent, nil, nil)
	ck(t, <-ch, bari.ObjectKeyEvent, nil, nil)
	ck(t, <-ch, bari.StringEvent, "foo", nil)
	ck(t, <-ch, bari.ObjectValueEvent, nil, nil)
	ck(t, <-ch, bari.NumberEvent, 10.0, nil)
}

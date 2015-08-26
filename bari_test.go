package bari_test

import (
	"fmt"
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

type expectedEvent struct {
	typ   bari.EventType
	value interface{}
	err   error
}

type testCase struct {
	data   string
	events []expectedEvent
}

var testCases = []testCase{
	{
		`{}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": "bar"}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.StringEvent, "bar", nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": "\u265e\u2602"}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.StringEvent, "♞☂", nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": 10}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.NumberEvent, int64(10), nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": 10.0}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.NumberEvent, float64(10), nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": true}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.BooleanEvent, true, nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": false}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.BooleanEvent, false, nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	// {
	// 	`{"foo": ["a", "b"]}`,
	// 	[]expectedEvent{
	// 		{bari.ObjectStartEvent, nil, nil},
	// 		{bari.ObjectKeyEvent, nil, nil},
	// 		{bari.StringEvent, "foo", nil},
	// 		{bari.ObjectValueEvent, nil, nil},
	// 		{bari.ArrayStartEvent, false, nil},
	// 		{bari.StringEvent, "a", nil},
	// 		{bari.StringEvent, "b", nil},
	// 		{bari.ArrayEndEvent, false, nil},
	// 		{bari.ObjectEndEvent, false, nil},
	// 	},
	// },

	// Invalid test cases

	{
		`{f}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.EOFEvent, nil, bari.ParseError{"expected \" but got f", 1, 2}},
		},
	},
	{
		`{"`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.EOFEvent, nil, bari.ParseError{"unexpected end of file", 1, 2}},
		},
	},
}

func TestParse(t *testing.T) {
	for _, c := range testCases {
		parser := bari.NewParser(strings.NewReader(c.data))
		ch := make(chan bari.Event)

		go func() {
			parser.Parse(ch)
			close(ch)
		}()

		for _, evt := range c.events {
			ev := <-ch
			fmt.Printf("%+v\n", ev)
			ck(t, ev, evt.typ, evt.value, evt.err)
		}
	}
}

package bari_test

import (
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
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
		`{"foo": 10e6}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.NumberEvent, float64(10e6), nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": -1.3}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.NumberEvent, float64(-1.3), nil},
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
	{
		`{"foo": []}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.ArrayStartEvent, nil, nil},
			{bari.ArrayEndEvent, nil, nil},
		},
	},
	{
		`{"foo": ["a", "b"]}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.ArrayStartEvent, nil, nil},
			{bari.StringEvent, "a", nil},
			{bari.StringEvent, "b", nil},
			{bari.ArrayEndEvent, nil, nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": "bar", "qux": "baz"}}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.StringEvent, "bar", nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "qux", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.StringEvent, "baz", nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
	{
		`{"foo": [{"a": true, "b": false}, {"b": 10.0, "c": [1, 2, 3]}]}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.ArrayStartEvent, nil, nil},

			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "a", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.BooleanEvent, true, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "b", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.BooleanEvent, false, nil},
			{bari.ObjectEndEvent, nil, nil},

			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "b", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.NumberEvent, float64(10), nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "c", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.ArrayStartEvent, nil, nil},
			{bari.NumberEvent, int64(1), nil},
			{bari.NumberEvent, int64(2), nil},
			{bari.NumberEvent, int64(3), nil},
			{bari.ArrayEndEvent, nil, nil},
			{bari.ObjectEndEvent, nil, nil},

			{bari.ArrayEndEvent, nil, nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},

	// Invalid test cases

	{
		``,
		[]expectedEvent{
			{bari.EOFEvent, nil, bari.ParseError{"unexpected end of file", 1, 0}},
		},
	},
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
	{
		`a`,
		[]expectedEvent{
			{bari.EOFEvent, nil, bari.ParseError{"unexpected character a", 1, 1}},
		},
	},
	{
		`[`,
		[]expectedEvent{
			{bari.ArrayStartEvent, nil, nil},
			{bari.EOFEvent, nil, bari.ParseError{"unexpected end of file", 1, 1}},
		},
	},
	{
		`["a"`,
		[]expectedEvent{
			{bari.ArrayStartEvent, nil, nil},
			{bari.StringEvent, "a", nil},
			{bari.EOFEvent, nil, bari.ParseError{"unexpected end of file", 1, 4}},
		},
	},
	{
		`["a", `,
		[]expectedEvent{
			{bari.ArrayStartEvent, nil, nil},
			{bari.StringEvent, "a", nil},
			{bari.EOFEvent, nil, bari.ParseError{"unexpected end of file", 1, 6}},
		},
	},

	// Multi object stream

	{
		`{"foo": "bar"}       {"bar": "baz"}`,
		[]expectedEvent{
			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "foo", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.StringEvent, "bar", nil},
			{bari.ObjectEndEvent, nil, nil},

			{bari.ObjectStartEvent, nil, nil},
			{bari.ObjectKeyEvent, nil, nil},
			{bari.StringEvent, "bar", nil},
			{bari.ObjectValueEvent, nil, nil},
			{bari.StringEvent, "baz", nil},
			{bari.ObjectEndEvent, nil, nil},
		},
	},
}

func TestParse(t *testing.T) {
	for i, c := range testCases {
		parser := bari.NewParser(strings.NewReader(c.data))
		ch := make(chan bari.Event)

		go func() {
			parser.Parse(ch)
			close(ch)
		}()

		for _, evt := range c.events {
			ev := <-ch
			fmt.Printf("case %d: `%s` %+v\n", i, c.data, ev)
			ck(t, ev, evt.typ, evt.value, evt.err)
		}
	}
}

func TestParseTestdata(t *testing.T) {
	f, err := os.Open("./testdata/code.json.gz")
	require.Nil(t, err)

	gz, err := gzip.NewReader(f)
	require.Nil(t, err)

	parser := bari.NewParser(gz)
	ch := make(chan bari.Event)

	go func() {
		parser.Parse(ch)
		close(ch)
	}()

	for ev := range ch {
		require.Nil(t, ev.Error)
	}
}

type cyclingReader struct {
	data string
	idx  int
}

func (r *cyclingReader) Read(b []byte) (n int, err error) {
	if len(b) >= len(r.data) {
		return copy(b, r.data), nil
	}

	n = copy(b, r.data[r.idx:])
	r.idx += n
	r.idx = r.idx % len(r.data)

	return n, nil
}

func TestCyclingReader(t *testing.T) {
	r := cyclingReader{data: `{"foo": "bar"}`}
	b := make([]byte, 5)

	n, _ := r.Read(b)
	require.Equal(t, 5, n)
	require.Equal(t, `{"foo`, string(b[:n]))

	n, _ = r.Read(b)
	require.Equal(t, 5, n)
	require.Equal(t, `": "b`, string(b[:n]))

	n, _ = r.Read(b)
	require.Equal(t, 4, n)
	require.Equal(t, `ar"}`, string(b[:n]))

	n, _ = r.Read(b)
	require.Equal(t, 5, n)
	require.Equal(t, `{"foo`, string(b[:n]))
}

func BenchmarkParseSimpleObject(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	const data = `{"foo": "bar"}`
	parser := bari.NewParser(&cyclingReader{data: data})
	ch := make(chan bari.Event)

	b.StartTimer()

	go func() {
		parser.Parse(ch)
	}()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 6; j++ {
			<-ch
		}
	}

	b.SetBytes(int64(len(data)))
}

func BenchmarkParseMultiObjectStream(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	const data = `{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}{"foo": "bar"}`
	parser := bari.NewParser(&cyclingReader{data: data})
	ch := make(chan bari.Event)

	b.StartTimer()

	go func() {
		parser.Parse(ch)
	}()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 6; j++ {
			<-ch
		}
	}

	b.SetBytes(int64(len(data)))
}

func BenchmarkParseStringWithUnicodeChars(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	const data = `{"foo": "\u265e\u2602"}`
	parser := bari.NewParser(&cyclingReader{data: data})
	ch := make(chan bari.Event)

	b.StartTimer()

	go func() {
		parser.Parse(ch)
	}()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 6; j++ {
			<-ch
		}
	}

	b.SetBytes(int64(len(data)))
}

func BenchmarkParseTestdata(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()

	f, err := os.Open("./testdata/code.json.gz")
	require.Nil(b, err)

	gz, err := gzip.NewReader(f)
	require.Nil(b, err)

	codeJSON, err := ioutil.ReadAll(gz)
	require.Nil(b, err)

	parser := bari.NewParser(&cyclingReader{data: string(codeJSON)})
	ch := make(chan bari.Event)

	b.StartTimer()

	go func() {
		parser.Parse(ch)
	}()

	for i := 0; i < b.N; i++ {
		for j := 0; j < 396995; j++ {
			<-ch
		}
	}

	b.SetBytes(int64(len(codeJSON)))
}

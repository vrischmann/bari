bari
====

[![Build Status](https://travis-ci.org/vrischmann/bari.svg?branch=master)](https://travis-ci.org/vrischmann/bari)
[![GoDoc](https://godoc.org/github.com/vrischmann/bari?status.svg)](https://godoc.org/github.com/vrischmann/bari)

bari is a JSON parser which works by emitting events for each interesting part of a JSON document.

For example, you will receive an event at the start of an object, for each string value, etc.

Use case
--------

It is intended to be used when you want to parse a big JSON file (think gigabytes plus) and you can't afford the memory needed to load this using the standard JSON parser.

Usage
-----

The parser works by emitting events into a channel. You are responsible for providing the channel, since you might want to buffer it.

Example code:

```go
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
```

The `NewParser` function takes a `io.Reader`, so it can read from a file, a network connection, or whatever else.

License
-------

bari is licensed under the MIT, see the LICENSE file.

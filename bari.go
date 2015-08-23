package bari

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
)

//go:generate stringer --type=EventType
type EventType uint

const (
	UnknownEvent EventType = iota
	ObjectStartEvent
	ObjectKeyEvent
	ObjectValueEvent
	ObjectEndEvent
	ArrayStartEvent
	ArrayEndEvent
	StringEvent
	NumberEvent
	BooleanEvent
	NullEvent
	EOFEvent
)

type Event struct {
	Type  EventType
	Value interface{}
	Error error
}

type Parser struct {
	br *bufio.Reader

	err error
	ch  chan Event
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		br: bufio.NewReader(r),
	}
}

var (
	eof = rune(0)

	errUnexpectedEOF = errors.New("unexpected eof")
)

func (p *Parser) Parse(ch chan Event) {
	p.ch = ch
	for {
		if !p.readObject() {
			break
		}
	}

	if err := p.getError(); err != nil {
		p.emitEvent(EOFEvent, nil, err)
	}
}

func (p *Parser) readObject() bool {
	var key string
	var value interface{}
	var ok bool

	r := p.readIgnoreWS()
	if r == eof {
		return false
	}

	if r != '{' {
		p.err = fmt.Errorf("expected { but got %c", r)
		return false
	}

	{
		p.emitEvent(ObjectStartEvent, nil, nil)

		key, ok = p.readString()
		if !ok {
			return false
		}

		p.emitEvent(ObjectKeyEvent, key, nil)
	}

	r = p.readIgnoreWS()
	if r != ':' {
		p.err = fmt.Errorf("expected : but got %c", r)
		return false
	}

	{
		value, ok = p.readValue()
		if !ok {
			return false
		}

		p.emitEvent(ObjectValueEvent, value, nil)
	}

	r = p.readIgnoreWS()
	if r != '}' {
		p.err = fmt.Errorf("expected } but got %c", r)
		return false
	}

	p.emitEvent(ObjectEndEvent, nil, nil)

	return true
}

func (p *Parser) getError() error {
	if p.err == io.EOF {
		return nil
	}

	return p.err
}

func (p *Parser) readString() (string, bool) {
	r := p.readIgnoreWS()
	if r == eof {
		p.err = errUnexpectedEOF
		return "", false
	}

	if r != '"' {
		p.err = fmt.Errorf("expected \" but got %c", r)
		return "", false
	}

	var buf bytes.Buffer
	for {
		r = p.readRune()
		if r == eof {
			p.err = errUnexpectedEOF
			return "", false
		}

		if r == '"' {
			break
		}

		buf.WriteRune(r)
	}

	return buf.String(), true
}

func (p *Parser) readValue() (interface{}, bool) {
	r := p.readIgnoreWS()
	if r == eof {
		p.err = errUnexpectedEOF
		return nil, false
	}

	switch r {
	case '"':
		p.unreadRune()
		val, ok := p.readString()
		return val, ok
	}

	return nil, false
}

func (p *Parser) readIgnoreWS() rune {
	r := p.readRune()
	for r != eof && unicode.IsSpace(r) {
		// eat whitespaces

		r = p.readRune()
	}
	return r
}

func (p *Parser) unreadRune() {
	p.br.UnreadRune()
}

func (p *Parser) readRune() rune {
	r, _, err := p.br.ReadRune()
	if err != nil {
		return eof
	}

	return r
}

func (p *Parser) emitEvent(typ EventType, value interface{}, err error) {
	p.ch <- Event{typ, value, err}
}

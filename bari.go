package bari

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
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

// TODO(vincent): change this API
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
	r := p.readIgnoreWS()
	if r == eof {
		return false
	}

	if r != '{' {
		p.err = fmt.Errorf("expected { but got %c", r)
		return false
	}

	p.emitEvent(ObjectStartEvent, nil, nil)

	{
		p.emitEvent(ObjectKeyEvent, nil, nil)

		ok := p.readString()
		if !ok {
			return false
		}
	}

	r = p.readIgnoreWS()
	if r != ':' {
		p.err = fmt.Errorf("expected : but got %c", r)
		return false
	}

	{
		p.emitEvent(ObjectValueEvent, nil, nil)

		ok := p.readValue()
		if !ok {
			return false
		}
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

func (p *Parser) readValue() bool {
	r := p.readIgnoreWS()
	if r == eof {
		p.err = errUnexpectedEOF
		return false
	}

	switch {
	case r == '"':
		p.unreadRune()
		return p.readString()
	case r == '\'':
		r := p.readRune()
		if r == eof {
			return false
		}

		return true
	case unicode.IsDigit(r):
		p.unreadRune()
		return p.readNumber()
	}

	return false
}

func (p *Parser) readNumber() bool {
	var buf bytes.Buffer
	isFloat := false
	for {
		r := p.readRune()
		if r == eof {
			p.err = errUnexpectedEOF
			return false
		}

		if r == '.' || r == 'e' {
			isFloat = true
		}

		if r != '.' && r != 'e' && r != '+' && r != '-' && !unicode.IsDigit(r) {
			p.unreadRune()
			break
		}

		buf.WriteRune(r)
	}

	if isFloat {
		f, err := strconv.ParseFloat(buf.String(), 64)
		if err != nil {
			p.err = err
			return false
		}

		p.emitEvent(NumberEvent, f, nil)

		return true
	}

	i, err := strconv.ParseInt(buf.String(), 10, 64)
	if err != nil {
		p.err = err
		return false
	}

	p.emitEvent(NumberEvent, i, nil)

	return true
}

// TODO(vincent): handle UTF-8 encoded strings
func (p *Parser) readString() bool {
	r := p.readIgnoreWS()
	if r == eof {
		p.err = errUnexpectedEOF
		return false
	}

	if r != '"' {
		p.err = fmt.Errorf("expected \" but got %c", r)
		return false
	}

	var buf bytes.Buffer
	for {
		r = p.readRune()
		if r == eof {
			p.err = errUnexpectedEOF
			return false
		}

		if r == '"' {
			break
		}

		buf.WriteRune(r)
	}

	p.emitEvent(StringEvent, buf.String(), nil)

	return true
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

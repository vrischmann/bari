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

	unreadChangesLine bool
	line              int
	position          int
}

type ParseError struct {
	Message  string
	Line     int
	Position int
}

func (p ParseError) Error() string {
	return fmt.Sprintf("ParseError: l:%d pos:%d msg:%s", p.Line, p.Position, p.Message)
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		br:   bufio.NewReader(r),
		line: 1,
	}
}

var (
	eof = rune(0)

	errUnexpectedEOF = errors.New("unexpected end of file")
)

func (p *Parser) Parse(ch chan Event) {
	p.ch = ch
	for {
		if !p.readObject() {
			break
		}

		p.resetState()
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
		p.serr("expected { but got %c", r)
		return false
	}

	p.emitEvent(ObjectStartEvent, nil, nil)

	r = p.readIgnoreWS()
	if r == '}' {
		p.emitEvent(ObjectEndEvent, nil, nil)
		return true
	}
	p.unreadRune()

	{
		p.emitEvent(ObjectKeyEvent, nil, nil)

		ok := p.readString()
		if !ok {
			return false
		}
	}

	r = p.readIgnoreWS()
	if r != ':' {
		p.serr("expected : but got %c", r)
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
		p.serr("expected } but got %c", r)
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
		p.serr2(errUnexpectedEOF)
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
	case r == 'f' || r == 't':
		p.unreadRune()
		return p.readBoolean()
	case unicode.IsDigit(r):
		p.unreadRune()
		return p.readNumber()
	}

	return false
}

func (p *Parser) readBoolean() bool {
	var buf bytes.Buffer

	for i := 0; i < 4; i++ {
		r := p.readRune()
		if r == eof {
			p.serr2(errUnexpectedEOF)
			return false
		}

		buf.WriteRune(r)
	}

	if buf.String() == "true" {
		p.emitEvent(BooleanEvent, true, nil)
		return true
	}

	r := p.readRune()
	if r == eof {
		p.serr2(errUnexpectedEOF)
		return false
	}

	if r != 'e' {
		p.serr("expected e but got %c", r)
		return false
	}

	p.emitEvent(BooleanEvent, false, nil)

	return true
}

func (p *Parser) readNumber() bool {
	var buf bytes.Buffer
	isFloat := false
	for {
		r := p.readRune()
		if r == eof {
			p.serr2(errUnexpectedEOF)
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
			p.serr2(err)
			return false
		}

		p.emitEvent(NumberEvent, f, nil)

		return true
	}

	i, err := strconv.ParseInt(buf.String(), 10, 64)
	if err != nil {
		p.serr2(err)
		return false
	}

	p.emitEvent(NumberEvent, i, nil)

	return true
}

// TODO(vincent): handle UTF-8 encoded strings
func (p *Parser) readString() bool {
	r := p.readIgnoreWS()
	if r == eof {
		p.serr2(errUnexpectedEOF)
		return false
	}

	if r != '"' {
		p.serr("expected \" but got %c", r)
		return false
	}

	var buf bytes.Buffer
	for {
		r = p.readRune()
		if r == eof {
			p.serr2(errUnexpectedEOF)
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
	p.position--
	if p.unreadChangesLine {
		p.line--
		p.position = 0
	}
	p.br.UnreadRune()
}

func (p *Parser) readRune() rune {
	r, _, err := p.br.ReadRune()
	if err != nil {
		p.err = err
		return eof
	}

	p.position++
	if r == '\n' {
		p.line++
		p.position = 0
		p.unreadChangesLine = true
	} else {
		p.unreadChangesLine = false
	}

	return r
}

func (p *Parser) emitEvent(typ EventType, value interface{}, err error) {
	p.ch <- Event{typ, value, err}
}

func (p *Parser) serr(format string, args ...interface{}) {
	p.err = ParseError{
		Message:  fmt.Sprintf(format, args...),
		Line:     p.line,
		Position: p.position,
	}
}

func (p *Parser) serr2(err error) {
	p.err = ParseError{
		Message:  err.Error(),
		Line:     p.line,
		Position: p.position,
	}
}

func (p *Parser) resetState() {
	p.line = 1
	p.position = 0
}

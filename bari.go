package bari

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
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
		p.serr2(errUnexpectedEOF)
		return false
	}

	if r != '{' {
		p.serr("expected { but got %c", r)
		return false
	}

	p.emitEvent(ObjectStartEvent, nil, nil)

	r = p.readIgnoreWS()
	if r == eof {
		p.serr2(errUnexpectedEOF)
		return false
	}

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

	decoded, ok := decodeToUTF8(buf.Bytes())
	if !ok {
		p.serr("unable to decode string into a valid UTF-8 string")
		return false
	}

	p.emitEvent(StringEvent, string(decoded), nil)

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

// this is taken from the Golang distribution.
// https://github.com/golang/go/blob/master/src/encoding/json/decode.go#L981-L1093
func decodeToUTF8(s []byte) (t []byte, ok bool) {
	// Check for unusual characters. If there are none,
	// then no unquoting is needed, so return a slice of the
	// original bytes.
	r := 0
	for r < len(s) {
		c := s[r]
		if c == '\\' || c == '"' || c < ' ' {
			break
		}
		if c < utf8.RuneSelf {
			r++
			continue
		}
		rr, size := utf8.DecodeRune(s[r:])
		if rr == utf8.RuneError && size == 1 {
			break
		}
		r += size
	}
	if r == len(s) {
		return s, true
	}

	b := make([]byte, len(s)+2*utf8.UTFMax)
	w := copy(b, s[0:r])
	for r < len(s) {
		// Out of room?  Can only happen if s is full of
		// malformed UTF-8 and we're replacing each
		// byte with RuneError.
		if w >= len(b)-2*utf8.UTFMax {
			nb := make([]byte, (len(b)+utf8.UTFMax)*2)
			copy(nb, b[0:w])
			b = nb
		}
		switch c := s[r]; {
		case c == '\\':
			r++
			if r >= len(s) {
				return
			}
			switch s[r] {
			default:
				return
			case '"', '\\', '/', '\'':
				b[w] = s[r]
				r++
				w++
			case 'b':
				b[w] = '\b'
				r++
				w++
			case 'f':
				b[w] = '\f'
				r++
				w++
			case 'n':
				b[w] = '\n'
				r++
				w++
			case 'r':
				b[w] = '\r'
				r++
				w++
			case 't':
				b[w] = '\t'
				r++
				w++
			case 'u':
				r--
				rr := getu4(s[r:])
				if rr < 0 {
					return
				}
				r += 6
				if utf16.IsSurrogate(rr) {
					rr1 := getu4(s[r:])
					if dec := utf16.DecodeRune(rr, rr1); dec != unicode.ReplacementChar {
						// A valid pair; consume.
						r += 6
						w += utf8.EncodeRune(b[w:], dec)
						break
					}
					// Invalid surrogate; fall back to replacement rune.
					rr = unicode.ReplacementChar
				}
				w += utf8.EncodeRune(b[w:], rr)
			}

		// Quote, control characters are invalid.
		case c == '"', c < ' ':
			return

		// ASCII
		case c < utf8.RuneSelf:
			b[w] = c
			r++
			w++

		// Coerce to well-formed UTF-8.
		default:
			rr, size := utf8.DecodeRune(s[r:])
			r += size
			w += utf8.EncodeRune(b[w:], rr)
		}
	}
	return b[0:w], true
}

// this is taken from the Golang distribution.
// https://github.com/golang/go/blob/master/src/encoding/json/decode.go#L960-L971
//
// getu4 decodes \uXXXX from the beginning of s, returning the hex value, or it returns -1.
func getu4(s []byte) rune {
	if len(s) < 6 || s[0] != '\\' || s[1] != 'u' {
		return -1
	}
	r, err := strconv.ParseUint(string(s[2:6]), 16, 64)
	if err != nil {
		return -1
	}
	return rune(r)
}

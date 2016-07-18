package bibtex

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

var field bool

// Scanner is a lexical scanner
type Scanner struct {
	r *bufio.Reader
}

// NewScanner returns a new instance of Scanner.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.eof is returned).
func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *Scanner) unread() {
	_ = s.r.UnreadRune()
}

// Scan returns the next token and literal value.
func (s *Scanner) Scan() (tok Token, lit string) {
	ch := s.read()
	if isWhitespace(ch) {
		s.ignoreWhitespace()
		ch = s.read()
	}
	if isAlphanum(ch) {
		s.unread()
		return s.scanIdent()
	}
	switch ch {
	case eof:
		return 0, ""
	case '@':
		return ATSIGN, string(ch)
	case ':':
		return COLON, string(ch)
	case ',':
		return COMMA, string(ch)
	case '=':
		field = true
		return EQUAL, string(ch)
	case '"':
		return s.scanQuoted()
	case '{':
		if field {
			defer func() { field = false }()
			return s.scanBraced()
		}
		return LBRACE, string(ch)
	case '}':
		return RBRACE, string(ch)
	case '#':
		return POUND, string(ch)
	case ' ':
		s.ignoreWhitespace()
	}

	log.Fatal(SyntaxError{What: fmt.Sprintf("Token %c unrecognised\n", ch)})
	return ILLEGAL, string(ch)
}

// scanIdent categorises a string to one of three categories.
func (s *Scanner) scanIdent() (tok Token, lit string) {
	switch ch := s.read(); ch {
	case '"':
		return s.scanQuoted()
	case '{':
		return s.scanBraced()
	default:
		s.unread() // Not open quote/brace.
		return s.scanBare()
	}
}

func (s *Scanner) scanBare() (Token, string) {
	var buf bytes.Buffer
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isAlphanum(ch) && !isBareSymbol(ch) || isWhitespace(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	str := buf.String()
	if strings.ToLower(str) == "comment" {
		return COMMENT, str
	} else if strings.ToLower(str) == "preamble" {
		return PREAMBLE, str
	} else if strings.ToLower(str) == "string" {
		return STRING, str
	} else if _, err := strconv.Atoi(str); err == nil { // Special case for numeric
		return IDENT, str
	}
	return BAREIDENT, str
}

// scanBraced parses a braced string, like {this}.
func (s *Scanner) scanBraced() (Token, string) {
	var buf bytes.Buffer
	var macro bool
	brace := 1
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '\\' {
			_, _ = buf.WriteRune(ch)
			macro = true
		} else if ch == '{' {
			_, _ = buf.WriteRune(ch)
			brace++
		} else if ch == '}' {
			brace--
			macro = false
			if brace == 0 { // Balances open brace.
				return IDENT, buf.String()
			}
			_, _ = buf.WriteRune(ch)
		} else if ch == '@' {
			if macro {
				_, _ = buf.WriteRune(ch)
			} else {
				log.Fatalf("%s: %s", ErrUnexpectedAtsign, buf.String())
			}
		} else if isWhitespace(ch) {
			_, _ = buf.WriteRune(ch)
			macro = false
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return ILLEGAL, buf.String()
}

// scanQuoted parses a quoted string, like "this".
func (s *Scanner) scanQuoted() (Token, string) {
	var buf bytes.Buffer
	brace := 0
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == '{' {
			brace++
		} else if ch == '}' {
			brace--
		} else if ch == '"' {
			if brace == 0 { // Matches open quote, unescaped
				return IDENT, buf.String()
			}
			_, _ = buf.WriteRune(ch)
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}
	return ILLEGAL, buf.String()
}

// ignoreWhitespace consumes the current rune and all contiguous whitespace.
func (s *Scanner) ignoreWhitespace() {
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		}
	}
}

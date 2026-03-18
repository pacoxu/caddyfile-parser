package caddyfileparser

import (
	"fmt"
	"io"
)

// Token represents a single token produced by the lexer.
type Token struct {
	// File is the source filename, used for error messages.
	File string

	// Line is the 1-based line number where the token appears.
	Line int

	// Text is the raw text of the token. For quoted strings the surrounding
	// quote characters are stripped and escape sequences are interpreted.
	Text string
}

// tokenize reads all configuration tokens from r and returns them in order.
// filename is used only for error reporting.
func tokenize(filename string, r io.Reader) ([]Token, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filename, err)
	}
	l := &lexer{filename: filename, buf: data, line: 1}
	return l.all()
}

// lexer holds the state needed to scan a configuration file byte-by-byte.
type lexer struct {
	filename string
	buf      []byte
	pos      int
	line     int
}

// all returns every token from the buffer.
func (l *lexer) all() ([]Token, error) {
	var tokens []Token
	for {
		tok, err := l.next()
		if err == io.EOF {
			return tokens, nil
		}
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
	}
}

// next returns the next meaningful token, skipping whitespace and comments.
func (l *lexer) next() (Token, error) {
	// Skip whitespace, counting newlines for line tracking.
	for l.pos < len(l.buf) {
		ch := l.buf[l.pos]
		switch ch {
		case '\n':
			l.line++
			l.pos++
		case ' ', '\t', '\r':
			l.pos++
		default:
			goto done
		}
	}
done:
	if l.pos >= len(l.buf) {
		return Token{}, io.EOF
	}

	ch := l.buf[l.pos]
	line := l.line

	// Single-line comment: skip everything through the newline.
	if ch == '#' {
		for l.pos < len(l.buf) && l.buf[l.pos] != '\n' {
			l.pos++
		}
		return l.next()
	}

	// Single-character structural tokens.
	if ch == '{' || ch == '}' || ch == ';' {
		l.pos++
		return Token{File: l.filename, Line: line, Text: string(ch)}, nil
	}

	// Quoted string.
	if ch == '"' || ch == '\'' {
		return l.readQuoted(line)
	}

	// Ordinary word token.
	return l.readWord(line)
}

// readQuoted scans a single- or double-quoted string, interpreting the common
// backslash escape sequences.  The surrounding quotes are not included in the
// returned token text.
func (l *lexer) readQuoted(startLine int) (Token, error) {
	quote := l.buf[l.pos]
	l.pos++ // skip opening quote

	var text []byte
	for l.pos < len(l.buf) {
		ch := l.buf[l.pos]
		switch {
		case ch == '\\' && l.pos+1 < len(l.buf):
			l.pos++
			switch l.buf[l.pos] {
			case 'n':
				text = append(text, '\n')
			case 't':
				text = append(text, '\t')
			case 'r':
				text = append(text, '\r')
			default:
				text = append(text, l.buf[l.pos])
			}
			l.pos++
		case ch == quote:
			l.pos++ // skip closing quote
			return Token{File: l.filename, Line: startLine, Text: string(text)}, nil
		case ch == '\n':
			l.line++
			text = append(text, ch)
			l.pos++
		default:
			text = append(text, ch)
			l.pos++
		}
	}
	return Token{}, fmt.Errorf("%s:%d: unterminated quoted string", l.filename, startLine)
}

// readWord scans a contiguous sequence of non-whitespace, non-special bytes.
func (l *lexer) readWord(startLine int) (Token, error) {
	start := l.pos
	for l.pos < len(l.buf) {
		ch := l.buf[l.pos]
		// Stop at whitespace or structural characters.
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
			ch == '{' || ch == '}' || ch == ';' || ch == '#' {
			break
		}
		l.pos++
	}
	if l.pos == start {
		return Token{}, io.EOF
	}
	return Token{File: l.filename, Line: startLine, Text: string(l.buf[start:l.pos])}, nil
}

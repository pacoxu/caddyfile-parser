package caddyfileparser

import (
	"fmt"
	"io"
)

// ParseNginx parses an Nginx configuration file from r and returns the
// top-level list of directives.
//
// The expected format uses semicolons to terminate simple directives and
// braces to delimit block directives, which may be arbitrarily nested:
//
//	worker_processes 4;
//
//	events {
//	    worker_connections 1024;
//	}
//
//	http {
//	    server {
//	        listen 80;
//	        server_name example.com;
//	    }
//	}
func ParseNginx(filename string, r io.Reader) (Config, error) {
	tokens, err := tokenize(filename, r)
	if err != nil {
		return nil, err
	}
	p := &nginxParser{tokens: tokens}
	return p.parseDirectives(false)
}

// nginxParser holds the parsing state for Nginx configuration format.
type nginxParser struct {
	tokens []Token
	pos    int
}

func (p *nginxParser) peek() (Token, bool) {
	if p.pos >= len(p.tokens) {
		return Token{}, false
	}
	return p.tokens[p.pos], true
}

func (p *nginxParser) consume() Token {
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

// parseDirectives parses a sequence of directives.  When inBlock is true the
// sequence is expected to end with a closing '}'; otherwise it runs until EOF.
func (p *nginxParser) parseDirectives(inBlock bool) ([]*Directive, error) {
	var directives []*Directive

	for {
		tok, ok := p.peek()
		if !ok {
			if inBlock {
				return nil, fmt.Errorf("unexpected end of file, expected '}'")
			}
			return directives, nil
		}

		if tok.Text == "}" {
			if !inBlock {
				return nil, fmt.Errorf("%s:%d: unexpected '}'", tok.File, tok.Line)
			}
			p.consume() // consume '}'
			return directives, nil
		}

		d, err := p.parseDirective()
		if err != nil {
			return nil, err
		}
		directives = append(directives, d)
	}
}

// parseDirective parses a single directive: a name, zero or more parameters,
// and either a terminating ';' or a brace-delimited body.
func (p *nginxParser) parseDirective() (*Directive, error) {
	nameTok, ok := p.peek()
	if !ok {
		return nil, fmt.Errorf("unexpected end of file")
	}
	if nameTok.Text == ";" || nameTok.Text == "{" || nameTok.Text == "}" {
		return nil, fmt.Errorf("%s:%d: unexpected %q", nameTok.File, nameTok.Line, nameTok.Text)
	}
	p.consume()

	d := &Directive{Name: nameTok.Text}

	// Collect parameters until ';' or '{'.
	for {
		tok, ok := p.peek()
		if !ok {
			return nil, fmt.Errorf("%s:%d: unexpected end of file after directive %q",
				nameTok.File, nameTok.Line, nameTok.Text)
		}

		switch tok.Text {
		case ";":
			p.consume()
			return d, nil
		case "{":
			p.consume()
			body, err := p.parseDirectives(true)
			if err != nil {
				return nil, err
			}
			d.Body = body
			return d, nil
		case "}":
			return nil, fmt.Errorf("%s:%d: unexpected '}' while reading directive %q",
				tok.File, tok.Line, nameTok.Text)
		default:
			d.Params = append(d.Params, p.consume().Text)
		}
	}
}

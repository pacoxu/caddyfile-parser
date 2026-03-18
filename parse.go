package caddyfileparser

import (
	"fmt"
	"io"
)

// Parse parses a Caddyfile or DNS Corefile from r and returns all server
// blocks it contains.
//
// The expected format is one or more labeled blocks:
//
//	label1 label2 {
//	    directive arg1 arg2
//	    directive2 {
//	        subdir value
//	    }
//	}
//
// Each directive occupies exactly one line; arguments are the remaining tokens
// on that same line.  An optional brace-delimited sub-block may follow the
// arguments on the same line.
func Parse(filename string, r io.Reader) ([]ServerBlock, error) {
	tokens, err := tokenize(filename, r)
	if err != nil {
		return nil, err
	}
	p := &caddyParser{tokens: tokens}
	return p.parseBlocks()
}

// caddyParser holds the parsing state for Caddyfile / Corefile format.
type caddyParser struct {
	tokens []Token
	pos    int
}

func (p *caddyParser) peek() (Token, bool) {
	if p.pos >= len(p.tokens) {
		return Token{}, false
	}
	return p.tokens[p.pos], true
}

func (p *caddyParser) consume() Token {
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}

// parseBlocks parses all top-level server blocks until EOF.
func (p *caddyParser) parseBlocks() ([]ServerBlock, error) {
	var blocks []ServerBlock

	for p.pos < len(p.tokens) {
		// Collect label keys until we see the opening '{'.
		var keys []string
		foundBrace := false
		for p.pos < len(p.tokens) {
			tok := p.tokens[p.pos]
			if tok.Text == "{" {
				p.pos++
				foundBrace = true
				break
			}
			if tok.Text == "}" || tok.Text == ";" {
				return nil, fmt.Errorf("%s:%d: unexpected %q", tok.File, tok.Line, tok.Text)
			}
			p.pos++
			keys = append(keys, tok.Text)
		}

		if len(keys) == 0 {
			break
		}
		if !foundBrace {
			return nil, fmt.Errorf("%s: unexpected end of file after keys %v, expected '{'",
				p.tokens[p.pos-1].File, keys)
		}

		segments, err := p.parseSegments()
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, ServerBlock{Keys: keys, Segments: segments})
	}

	return blocks, nil
}

// parseSegments parses directives until it finds the matching closing '}'.
func (p *caddyParser) parseSegments() ([]Segment, error) {
	var segments []Segment

	for p.pos < len(p.tokens) {
		tok, ok := p.peek()
		if !ok {
			return nil, fmt.Errorf("unexpected end of file, expected '}'")
		}

		// End of this block.
		if tok.Text == "}" {
			p.consume()
			return segments, nil
		}

		// Structural characters that are not valid as directive names here.
		if tok.Text == "{" || tok.Text == ";" {
			return nil, fmt.Errorf("%s:%d: unexpected %q", tok.File, tok.Line, tok.Text)
		}

		// Directive name is the first token on the line.
		nameTok := p.consume()
		seg := Segment{Name: nameTok.Text}

		// Collect arguments: remaining tokens on the same line.
		// If '{' appears on the same line, open a sub-block instead.
		for p.pos < len(p.tokens) {
			next, _ := p.peek()

			// End of block: stop collecting args.
			if next.Text == "}" {
				break
			}

			// Opening sub-block on the same line as the directive.
			if next.Text == "{" {
				if next.Line != nameTok.Line {
					// '{' on a different line belongs to a new server block;
					// stop collecting args for this directive.
					break
				}
				p.consume() // consume '{'
				sub, err := p.parseSegments()
				if err != nil {
					return nil, err
				}
				seg.Block = sub
				break
			}

			// Token on a different line → start of the next directive.
			if next.Line != nameTok.Line {
				break
			}

			seg.Args = append(seg.Args, p.consume().Text)
		}

		segments = append(segments, seg)
	}

	return nil, fmt.Errorf("unexpected end of file, expected '}'")
}

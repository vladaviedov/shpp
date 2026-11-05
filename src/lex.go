/**
 * @file lex.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief File scanner and token lexer.
 */
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type TokenKind uint64

const (
	// Special
	tError TokenKind = iota
	tEOF
	tPreambleEnd
	// Blobs
	tDirectiveName
	tDirectiveArgument
	tRaw
)

type QuoteState uint64

const (
	qNone QuoteState = iota
	qSingle
	qDouble
)

type Token struct {
	Kind TokenKind
	Data string
}

// Premade (const) special tokens
var (
	tokenError       = Token{Kind: tError, Data: ""}
	tokenEOF         = Token{Kind: tEOF, Data: ""}
	tokenPreambleEnd = Token{Kind: tPreambleEnd, Data: ""}
	tokenEscape      = Token{Kind: tRaw, Data: "@"}
)

var advanceEOF = fmt.Errorf("failed to read stream: %s", io.EOF)

type Lexer struct {
	Stream *bufio.Reader

	// Current character
	Head rune

	// State flags
	FlagPreamble  bool
	FlagDirective bool
	FlagStrong    bool
	FlagEOF       bool
}

func newLexer(source io.Reader) (*Lexer, error) {
	stream := bufio.NewReader(source)
	head, _, err := stream.ReadRune()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize lexer: %w", err)
	}

	s := &Lexer{
		Stream:        stream,
		Head:          head,
		FlagPreamble:  true,
		FlagDirective: false,
		FlagEOF:       false,
	}

	return s, nil
}

func (s *Lexer) nextToken() (Token, error) {
	if s.FlagEOF {
		return tokenEOF, nil
	}

	var rawBuf strings.Builder

	for {
		if s.FlagDirective {
			// Prepare lookahead
			var next byte
			nextRes, err := s.Stream.Peek(1)
			if err != nil {
				if err != io.EOF {
					return tokenError, fmt.Errorf("failed to read stream: %w", err)
				}

				next = '\n'
			} else {
				next = nextRes[0]
			}

			switch s.Head {
			case ' ':
			case '\t':
			case '@':
				// Exit directive mode
				s.FlagDirective = false
			case '\n':
				// Exit directive mode
				s.FlagDirective = false
				if !s.FlagPreamble {
					rawBuf.WriteRune(s.Head)
				}
			case '\\':
				// Check line continuation
				if next == '\n' {
					s.Stream.Discard(1)
				}
			default:
				// Parse argument
				arg, err := s.parseDirectiveArgument()
				if err != nil {
					return tokenError, err
				}

				return Token{
					Kind: tDirectiveArgument,
					Data: arg,
				}, nil
			}
		} else {
			switch s.Head {
			case '@':
				// If HTML buffer has data, flush now
				if rawBuf.Len() > 0 {
					return Token{
						Kind: tRaw,
						Data: rawBuf.String(),
					}, nil
				}

				// Enter directive mode
				s.FlagDirective = true

				// Parse directive name
				dir, err := s.parseDirectiveName()
				if err != nil {
					return tokenError, err
				}

				// Null directive - escaped @
				if dir == "" {
					err = s.advanceHead()
					if err != nil {
						return tokenError, err
					}
					return tokenEscape, nil
				}

				// Strong mode
				if dir[0] == '!' {
					dir = dir[1:]
					s.FlagStrong = true
				} else {
					s.FlagStrong = false
				}

				// Regular directive name
				return Token{
					Kind: tDirectiveName,
					Data: dir,
				}, nil
			case '<':
				// End preamble - do not consume char
				if s.FlagPreamble {
					s.FlagPreamble = false
					return tokenPreambleEnd, nil
				}
				// Otherwise treat as normal
				fallthrough
			default:
				// Save to buffer
				if !s.FlagPreamble {
					rawBuf.WriteRune(s.Head)
				}
			}
		}

		// No token emitted from current head - continue
		err := s.advanceHead()
		if err != nil {
			// Note: EOF is allowed in raw mode (outside preamble)
			if !s.FlagPreamble && !s.FlagDirective && err.Error() == advanceEOF.Error() {
				s.FlagEOF = true

				// Need to flush buffer before actual EOF
				if rawBuf.Len() > 0 {
					return Token{
						Kind: tRaw,
						Data: rawBuf.String(),
					}, nil
				}

				return tokenEOF, nil
			}

			return tokenError, err
		}
	}
}

func (s *Lexer) parseDirectiveName() (string, error) {
	var parsed strings.Builder

	// Advance head at the start
	err := s.advanceHead()
	if err != nil {
		return "", err
	}

	for s.Head != ' ' && s.Head != '\t' && s.Head != '\n' {
		if !unicode.IsLetter(s.Head) && s.Head != '!' {
			return "", fmt.Errorf("failed to lex directive: invalid character %q", s.Head)
		}
		parsed.WriteRune(s.Head)

		err := s.advanceHead()
		if err != nil {
			return "", err
		}
	}

	return parsed.String(), nil
}

func (s *Lexer) parseDirectiveArgument() (string, error) {
	var parsed strings.Builder

	state := qNone
	for {
		switch s.Head {
		case '"':
			switch state {
			case qNone:
				state = qDouble
			case qSingle:
				parsed.WriteRune(s.Head)
			case qDouble:
				state = qNone
			}
		case '\'':
			switch state {
			case qNone:
				state = qSingle
			case qSingle:
				state = qNone
			case qDouble:
				parsed.WriteRune(s.Head)
			}
		case '{':
			if state == qNone {
				ident, err := s.parseIdentifier()
				if err != nil {
					return "", err
				}

				parsed.WriteRune('{')
				parsed.WriteString(ident)
				parsed.WriteRune('}')
			} else {
				parsed.WriteRune(s.Head)
			}
		case ' ':
			// Ignore WS if argument is empty
			if parsed.Len() == 0 {
				break
			}
			fallthrough
		case '@':
			fallthrough
		case '\t':
			if state != qNone {
				parsed.WriteRune(s.Head)
				break
			}

			fallthrough
		case '\n':
			// Open quote
			if state != qNone {
				return "", fmt.Errorf("syntax error: unclosed quotes")
			}
			return parsed.String(), nil
		case '\\':
			err := s.advanceHead()
			if err != nil {
				return "", err
			}

			switch s.Head {
			case '"':
				if state == qSingle {
					parsed.WriteRune('\\')
				}
				parsed.WriteRune(s.Head)
			case '\'':
				if state == qDouble {
					parsed.WriteRune('\\')
				}
				parsed.WriteRune(s.Head)
			case '\\':
				parsed.WriteRune(s.Head)
			case ' ':
				fallthrough
			case '\t':
				fallthrough
			case '\n':
				return "", errors.New("failed to lex argument: cannot escape whitespace")
			default:
				parsed.WriteRune('\\')
				parsed.WriteRune(s.Head)
			}
		default:
			parsed.WriteRune(s.Head)
		}

		err := s.advanceHead()
		if err != nil {
			return "", err
		}
	}
}

func (s *Lexer) parseIdentifier() (string, error) {
	var parsed strings.Builder
	numberIdent := false

	// Advance head at the start
	err := s.advanceHead()
	if err != nil {
		return "", err
	}

	for s.Head != '}' {
		if parsed.Len() == 0 && unicode.IsNumber(s.Head) {
			parsed.WriteRune(s.Head)
			numberIdent = true
		} else {
			if numberIdent && !unicode.IsNumber(s.Head) {
				return "", fmt.Errorf("failed to lex identifier: non-digit in numerical argument")
			}

			parsed.WriteRune(s.Head)
		}

		err := s.advanceHead()
		if err != nil {
			return "", err
		}
	}

	return parsed.String(), nil
}

func (s *Lexer) advanceHead() error {
	val, _, err := s.Stream.ReadRune()
	if err != nil {
		return fmt.Errorf("failed to read stream: %w", err)
	}

	if s.FlagDirective && s.FlagStrong && val == '\n' {
		val = ' '
	}
	s.Head = val
	return nil
}

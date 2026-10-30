/**
 * @file lexer.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief File scanner and token lexer.
 */
package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
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

type Lexer struct {
	Stream *bufio.Reader

	// Current character
	Head byte

	// State flags
	FlagPreamble  bool
	FlagDirective bool
	FlagEOF       bool
}

func NewLexer(source io.Reader) (*Lexer, error) {
	stream := bufio.NewReader(source)
	head, err := stream.ReadByte()
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

func (s *Lexer) NextToken() Token {
	if s.FlagEOF {
		return tokenEOF
	}

	var rawBuf strings.Builder

	for {
		if s.FlagDirective {
			// Prepare lookahead
			nextRes, err := s.Stream.Peek(1)
			if err != nil {
				return tokenError
			}
			next := nextRes[0]

			switch s.Head {
			case '@':
				fallthrough
			case '\n':
				// Exit directive mode
				s.FlagDirective = false
				if !s.FlagPreamble {
					rawBuf.WriteByte(s.Head)
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
					return tokenError
				}

				return Token{
					Kind: tDirectiveArgument,
					Data: arg,
				}
			}
		} else {
			switch s.Head {
			case '@':
				// If HTML buffer has data, flush now
				if rawBuf.Len() > 0 {
					return Token{
						Kind: tRaw,
						Data: rawBuf.String(),
					}
				}

				// Enter directive mode
				s.FlagDirective = true

				// Parse directive name
				dir, err := s.parseDirectiveName()
				if err != nil {
					return tokenError
				}

				// Null directive - escaped @
				if dir == "" {
					if s.advanceHead() != nil {
						return tokenError
					}
					return tokenEscape
				}

				// Regular directive name
				return Token{
					Kind: tDirectiveName,
					Data: dir,
				}
			case '<':
				// End preamble - do not consume char
				if s.FlagPreamble {
					s.FlagPreamble = false
					return tokenPreambleEnd
				}
				// Otherwise treat as normal
				fallthrough
			default:
				// Save to buffer
				if !s.FlagPreamble {
					rawBuf.WriteByte(s.Head)
				}
			}
		}

		// No token emitted from current head - continue
		err := s.advanceHead()
		if err != nil {
			// Note: EOF is allowed in raw mode (outside preamble)
			if !s.FlagPreamble && !s.FlagDirective && err == io.EOF {
				s.FlagEOF = true

				// Need to flush buffer before actual EOF
				if rawBuf.Len() > 0 {
					return Token{
						Kind: tRaw,
						Data: rawBuf.String(),
					}
				}

				return tokenEOF
			}

			return tokenError
		}
	}
}

func (s *Lexer) parseDirectiveName() (string, error) {
	// TODO: implement
	return "", nil
}

func (s *Lexer) parseDirectiveArgument() (string, error) {
	var parsed strings.Builder

	err := s.advanceHead()
	if err != nil {
		return "", err
	}

	state := qNone
	for {
		switch (s.Head) {
		case '"':
			switch (state) {
			case qNone:
				state = qDouble
			case qSingle:
				parsed.WriteByte(s.Head)
			case qDouble:
				state = qNone
			}
		case '\'':
			switch (state) {
			case qNone:
				state = qSingle
			case qSingle:
				state = qNone
			case qDouble:
				parsed.WriteByte(s.Head)
			}
		case '{':
			if state == qNone {
				ident, err := s.parseIdentifier()
				if err != nil {
					return "", nil
				}

				parsed.WriteString(ident)
			} else {
				parsed.WriteByte(s.Head)
			}
		case ' ':
			fallthrough
		case '\t':
			fallthrough
		case '\n':
			return parsed.String(), nil
		case '\\':
			parsed.WriteByte(s.Head)
			err := s.advanceHead()
			if err != nil {
				return "", err
			}
			parsed.WriteByte(s.Head)
		}
	}
}

func (s *Lexer) parseIdentifier() (string, error) {
	// TODO: implement
	return "", nil
}

func (s *Lexer) advanceHead() error {
	val, err := s.Stream.ReadByte()
	if err != nil {
		return err
	}

	s.Head = val
	return nil
}

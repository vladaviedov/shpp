/**
 * @file lexer_test.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief File scanner and token lexer.
 */
package main

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

func TestNewLexer_InvalidStream(t *testing.T) {
	assert := assert.New(t)
	errStr := "mock error"
	reader := iotest.ErrReader(errors.New(errStr))

	lexer, err := NewLexer(reader)

	assert.Contains(err.Error(), errStr)
	assert.Nil(lexer)
}

func TestNewLexer_EmptyFile(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("")

	lexer, err := NewLexer(reader)

	assert.ErrorIs(io.EOF, err)
	assert.Nil(lexer)
}

func TestNewLexer_Success(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("a")

	lexer, err := NewLexer(reader)

	assert.Nil(err)
	assert.Equal(byte('a'), lexer.Head)
}

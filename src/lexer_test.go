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
	errStr := "mock error"
	reader := iotest.ErrReader(errors.New(errStr))

	lexer, err := NewLexer(reader)

	assert.Contains(t, err.Error(), errStr)
	assert.Nil(t, lexer)
}

func TestNewLexer_EmptyFile(t *testing.T) {
	reader := strings.NewReader("")

	lexer, err := NewLexer(reader)

	assert.ErrorIs(t, err, io.EOF)
	assert.Nil(t, lexer)
}

func TestNewLexer_Success(t *testing.T) {
	reader := strings.NewReader("a")

	lexer, err := NewLexer(reader)

	assert.Nil(t, err)
	assert.Equal(t, lexer.Head, byte('a'))
}

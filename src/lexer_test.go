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

	assert.ErrorIs(err, io.EOF)
	assert.Nil(lexer)
}

func TestNewLexer_Success(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("a")

	lexer, err := NewLexer(reader)

	assert.Nil(err)
	assert.Equal(rune('a'), lexer.Head)
}

const onlyRaw = `<html>
	<body></body>
</html>
`

func TestNextToken_OnlyRaw(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(onlyRaw)

	lexer, err := NewLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body></body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		assert.Equal(expect, lexer.NextToken())
	}
}

const preambleDirectives = `@ignore
@url index.html
@script script.js
<html>
	<body></body>
</html>
`

func TestNextToken_PreambleDirectives(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(preambleDirectives)

	lexer, err := NewLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tDirectiveName, Data: "ignore"},
		{Kind: tDirectiveName, Data: "url"},
		{Kind: tDirectiveArgument, Data: "index.html"},
		{Kind: tDirectiveName, Data: "script"},
		{Kind: tDirectiveArgument, Data: "script.js"},
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body></body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		assert.Equal(expect, lexer.NextToken())
	}
}

const bodyDirectives = `<html>
	<body>
		@include part.html
		@run date
	</body>
</html>
`

func TestNextToken_BodyDirectives(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(bodyDirectives)

	lexer, err := NewLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body>\n\t\t"},
		{Kind: tDirectiveName, Data: "include"},
		{Kind: tDirectiveArgument, Data: "part.html"},
		{Kind: tRaw, Data: "\n\t\t"},
		{Kind: tDirectiveName, Data: "run"},
		{Kind: tDirectiveArgument, Data: "date"},
		{Kind: tRaw, Data: "\n\t</body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		assert.Equal(expect, lexer.NextToken())
	}
}

const multiArgumentDirectives = `<html>
	@derive template.shpp.in arg1 arg2
</html>
`

func TestNextToken_MultiArgumentDirective(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(multiArgumentDirectives)

	lexer, err := NewLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t"},
		{Kind: tDirectiveName, Data: "derive"},
		{Kind: tDirectiveArgument, Data: "template.shpp.in"},
		{Kind: tDirectiveArgument, Data: "arg1"},
		{Kind: tDirectiveArgument, Data: "arg2"},
		{Kind: tRaw, Data: "\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		assert.Equal(expect, lexer.NextToken())
	}
}

const noBody = `@ignore
`

func TestNextToken_NoBody(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(noBody)

	lexer, err := NewLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tDirectiveName, Data: "ignore"},
		{Kind: tError, Data: ""},
	}

	for _, expect := range expectedList {
		assert.Equal(expect, lexer.NextToken())
	}
}

const closedDirectives = `<html>
	<body>
		Today is @run date +\"%Y-%m-%d\"@ and the time is
		@run date +"%H:%M:%S"@ in @run date +"%Z"@ timezone.
	</body>
</html>
`

func TestNextToken_ClosedDirectives(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(closedDirectives)

	lexer, err := NewLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body>\n\t\tToday is "},
		{Kind: tDirectiveName, Data: "run"},
		{Kind: tDirectiveArgument, Data: "date"},
		{Kind: tDirectiveArgument, Data: "+\"%Y-%m-%d\""},
		{Kind: tRaw, Data: " and the time is\n\t\t"},
		{Kind: tDirectiveName, Data: "run"},
		{Kind: tDirectiveArgument, Data: "date"},
		{Kind: tDirectiveArgument, Data: "+\"%H:%M:%S\""},
		{Kind: tRaw, Data: " in "},
		{Kind: tDirectiveName, Data: "run"},
		{Kind: tDirectiveArgument, Data: "date"},
		{Kind: tDirectiveArgument, Data: "+\"%Z\""},
		{Kind: tRaw, Data: " timezone.\n\t</body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		assert.Equal(expect, lexer.NextToken())
	}
}

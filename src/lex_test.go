/**
 * @file lex_test.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version 0.9.0
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

/* newLexer */

func TestNewLexer_InvalidStream(t *testing.T) {
	assert := assert.New(t)
	errStr := "mock error"
	reader := iotest.ErrReader(errors.New(errStr))

	lexer, err := newLexer(reader)

	assert.Contains(err.Error(), errStr)
	assert.Nil(lexer)
}

func TestNewLexer_EmptyFile(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("")

	lexer, err := newLexer(reader)

	assert.ErrorIs(err, io.EOF)
	assert.Nil(lexer)
}

func TestNewLexer_Success(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("a")

	lexer, err := newLexer(reader)

	assert.Nil(err)
	assert.Equal(rune('a'), lexer.Head)
}

const onlyRaw = `<html>
	<body></body>
</html>
`

/* nextToken */

func TestNextToken_OnlyRaw(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(onlyRaw)

	lexer, err := newLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body></body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
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

	lexer, err := newLexer(reader)
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
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
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

	lexer, err := newLexer(reader)
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
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
	}
}

const multiArgumentDirectives = `<html>
	@derive template.shpp.in arg1 arg2
</html>
`

func TestNextToken_MultiArgumentDirective(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(multiArgumentDirectives)

	lexer, err := newLexer(reader)
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
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
	}
}

const noBody = `@ignore
`

func TestNextToken_NoBody(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(noBody)

	lexer, err := newLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tDirectiveName, Data: "ignore"},
		{Kind: tError, Data: ""},
	}

	for _, expect := range expectedList {
		token, err := lexer.nextToken()

		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
	}
}

const closedDirectives = `<html>
	<body>
		Today is @run date +\"%Y-%m-%d\"@ and the time is
		@run date +\"%H:%M:%S\"@ in @run date +\"%Z\"@ timezone.
	</body>
</html>
`

func TestNextToken_ClosedDirectives(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(closedDirectives)

	lexer, err := newLexer(reader)
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
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
	}
}

const withEscapes = `<html>
	<body>
		@run echo \@ \" \' \\
	</body>
</html>
`

func TestNextToken_WithEscapes(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(withEscapes)

	lexer, err := newLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body>\n\t\t"},
		{Kind: tDirectiveName, Data: "run"},
		{Kind: tDirectiveArgument, Data: "echo"},
		{Kind: tDirectiveArgument, Data: "@"},
		{Kind: tDirectiveArgument, Data: "\""},
		{Kind: tDirectiveArgument, Data: "'"},
		{Kind: tDirectiveArgument, Data: "\\"},
		{Kind: tRaw, Data: "\n\t</body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
	}
}

const multilineDirective = `<html>
	<body>
		@derive template.shpp.in \
			with \
			"Second Argument"
	</body>
</html>
`

func TestNextToken_MultilineDirective(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(multilineDirective)

	lexer, err := newLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body>\n\t\t"},
		{Kind: tDirectiveName, Data: "derive"},
		{Kind: tDirectiveArgument, Data: "template.shpp.in"},
		{Kind: tDirectiveArgument, Data: "with"},
		{Kind: tDirectiveArgument, Data: "Second Argument"},
		{Kind: tRaw, Data: "\n\t</body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
	}
}

const strongMode = `<html>
	<body>
		@!derive template.shpp.in with "Second
Argument"@
	</body>
</html>
`

func TestNextToken_StrongMode(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(strongMode)

	lexer, err := newLexer(reader)
	assert.Nil(err)

	var expectedList = []Token{
		{Kind: tPreambleEnd, Data: ""},
		{Kind: tRaw, Data: "<html>\n\t<body>\n\t\t"},
		{Kind: tDirectiveName, Data: "derive"},
		{Kind: tDirectiveArgument, Data: "template.shpp.in"},
		{Kind: tDirectiveArgument, Data: "with"},
		{Kind: tDirectiveArgument, Data: "Second Argument"},
		{Kind: tRaw, Data: "\n\t</body>\n</html>\n"},
		{Kind: tEOF, Data: ""},
	}

	for _, expect := range expectedList {
		token, err := lexer.nextToken()
		if expect.Kind == tError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
		}
		assert.Equal(expect, token)
	}
}

/* parseDirectiveArgument */

func TestParseDirectiveArgument_StartEOF(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader(" ")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	_, err = lexer.parseDirectiveArgument()
	assert.NotNil(err)
	assert.Contains(err.Error(), io.EOF.Error())
}

func TestParseDirectiveArgument_Literal(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader(" test1 test2\n")
	reader2 := strings.NewReader(" test1 test2@")

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)

	arg, err := lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("test1", arg)
	arg, err = lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("test2", arg)

	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("test1", arg)
	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("test2", arg)
}

func TestParseDirectiveArgument_Quotes(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader(" double quotes \"double quotes\"\n")
	reader2 := strings.NewReader(" single quotes 'single quotes'\n")
	reader3 := strings.NewReader(" \"smushed\"\"double\"\n")
	reader4 := strings.NewReader(" 'smushed''single'\n")
	reader5 := strings.NewReader(" 'smushed'\"both\"\n")

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)
	lexer3, err := newLexer(reader3)
	assert.Nil(err)
	lexer4, err := newLexer(reader4)
	assert.Nil(err)
	lexer5, err := newLexer(reader5)
	assert.Nil(err)

	arg, err := lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("double", arg)
	arg, err = lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("quotes", arg)
	arg, err = lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("double quotes", arg)

	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("single", arg)
	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("quotes", arg)
	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("single quotes", arg)

	arg, err = lexer3.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("smusheddouble", arg)

	arg, err = lexer4.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("smushedsingle", arg)

	arg, err = lexer5.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("smushedboth", arg)
}

func TestParseDirectiveArgument_OpenQuotes(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader("\"double\n")
	reader2 := strings.NewReader("'single\n")

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)

	_, err = lexer1.parseDirectiveArgument()
	assert.NotNil(err)

	_, err = lexer2.parseDirectiveArgument()
	assert.NotNil(err)
}

func TestParseDirectiveArgument_ExtraWhitespace(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("test1  test2\n")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	arg, err := lexer.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("test1", arg)
	arg, err = lexer.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("test2", arg)
}

func TestParseDirectiveArgument_Empty(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("\n")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	arg, err := lexer.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("", arg)
}

func TestParseDirectiveArgument_Variables(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader("{test}\n")
	reader2 := strings.NewReader("{test1}{test2}\n")
	reader3 := strings.NewReader("{1badtest}\n")

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)
	lexer3, err := newLexer(reader3)
	assert.Nil(err)

	arg, err := lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("{test}", arg)

	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal("{test1}{test2}", arg)

	_, err = lexer3.parseDirectiveArgument()
	assert.NotNil(err)
}

func TestParseDirectiveArgument_GoodEscapes(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader(`\" \' \\ \@` + "\n")
	reader2 := strings.NewReader(`"\"" '\'' "\\" '\\'` + "\n")

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)

	arg, err := lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`"`, arg)
	arg, err = lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`'`, arg)
	arg, err = lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`\`, arg)
	arg, err = lexer1.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`@`, arg)

	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`"`, arg)
	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`'`, arg)
	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`\`, arg)
	arg, err = lexer2.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`\`, arg)
}

func TestParseDirectiveArgument_BadEscapes(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader("\\ \n")
	reader2 := strings.NewReader("\\\t\n")
	reader3 := strings.NewReader("\\\n\n")
	reader4 := strings.NewReader(`"\'" '\"' \a` + "\n")
	reader5 := strings.NewReader(`\`)

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)
	lexer3, err := newLexer(reader3)
	assert.Nil(err)
	lexer4, err := newLexer(reader4)
	assert.Nil(err)
	lexer5, err := newLexer(reader5)
	assert.Nil(err)

	_, err = lexer1.parseDirectiveArgument()
	assert.NotNil(err)

	_, err = lexer2.parseDirectiveArgument()
	assert.NotNil(err)

	_, err = lexer3.parseDirectiveArgument()
	assert.NotNil(err)

	arg, err := lexer4.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`\'`, arg)
	arg, err = lexer4.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`\"`, arg)
	arg, err = lexer4.parseDirectiveArgument()
	assert.Nil(err)
	assert.Equal(`\a`, arg)

	_, err = lexer5.parseDirectiveArgument()
	assert.NotNil(err)
}

func TestParseDirectiveArgument_EndEOF(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("test")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	_, err = lexer.parseDirectiveArgument()
	assert.NotNil(err)
	assert.Contains(err.Error(), io.EOF.Error())
}

/* parseIdentifier */

func TestParseIdentifier_StartEOF(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("{")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	_, err = lexer.parseIdentifier()
	assert.NotNil(err)
	assert.Contains(err.Error(), io.EOF.Error())
}

func TestParseIdentifier_Unterminated(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("{test\n")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	_, err = lexer.parseIdentifier()
	assert.NotNil(err)
}

func TestParseIdentifier_StringIdentifier(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("{test}\n")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	ident, err := lexer.parseIdentifier()
	assert.Nil(err)
	assert.Equal("test", ident)
}

func TestParseIdentifier_NumericalIdentifier(t *testing.T) {
	assert := assert.New(t)
	reader := strings.NewReader("{4}\n")

	lexer, err := newLexer(reader)
	assert.Nil(err)

	ident, err := lexer.parseIdentifier()
	assert.Nil(err)
	assert.Equal("4", ident)
}

func TestParseIdentifier_MixedIdentifier(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader("{2cool}\n")
	reader2 := strings.NewReader("{cool4}\n")

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)

	// Starting with int not allowed
	_, err = lexer1.parseIdentifier()
	assert.NotNil(err)

	// Continue with int allowed
	ident, err := lexer2.parseIdentifier()
	assert.Nil(err)
	assert.Equal("cool4", ident)
}

func TestParseIdentifier_SpecialChars(t *testing.T) {
	assert := assert.New(t)
	reader1 := strings.NewReader("{.name}\n")
	reader2 := strings.NewReader("{.line}\n")

	lexer1, err := newLexer(reader1)
	assert.Nil(err)
	lexer2, err := newLexer(reader2)
	assert.Nil(err)

	// Starting with int not allowed
	ident, err := lexer1.parseIdentifier()
	assert.Nil(err)
	assert.Equal(".name", ident)

	// Continue with int allowed
	ident, err = lexer2.parseIdentifier()
	assert.Nil(err)
	assert.Equal(".line", ident)
}

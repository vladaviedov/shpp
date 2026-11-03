/**
 * @file process.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief Input file processing logic.
 */
package main

import (
	"io"
)

type Asset struct {
	Kind       AssetKind
	SourcePath string
}

type State struct {
	Assets   []Asset
	PageURL  string
	FileDir  string
	ArgCount uint64
	ArgData  []string

	Emitter *Emitter

	UrlChanged bool
	IsTemplate bool
	Ignored    bool
}

func ProcessFile(inStream io.Reader, outStream io.Writer, defaultName string, workingDir string) error {
	state := &State{
		PageURL:  defaultName + ".html",
		FileDir:  workingDir,
		ArgCount: 0,
		ArgData:  nil,
		Emitter:  NewEmitter(outStream),
	}

	err := processFileIM(inStream, state)
	if err != nil {
		return err
	}

	return writeOutput(state)
}

func processFileIM(stream io.Reader, state *State) error {
	lexer, err := NewLexer(stream)
	if err != nil {
		return err
	}

	// Process preamble
	err = processPreamble(lexer, state)
	if err != nil {
		return err
	}
	if state.Ignored {
		return nil
	}

	// Process body
	return processBody(lexer, state)
}

func processPreamble(lexer *Lexer, state *State) error {
	token, err := lexer.NextToken()
	if err != nil {
		return err
	}

	var tokenList []Token
	for {
		switch token.Kind {
		case tDirectiveName:
			if len(tokenList) != 0 {
				// TODO: process directive
			}
			fallthrough
		case tDirectiveArgument:
			tokenList = append(tokenList, token)
		case tPreambleEnd:
			if len(tokenList) != 0 {
				// TODO: process directive
			}

			return nil
		}

		// Short circuit if ignore flag is raised
		if state.Ignored {
			return nil
		}

		token, err = lexer.NextToken()
		if err != nil {
			return err
		}
	}
}

func processBody(lexer *Lexer, state *State) error {
	token, err := lexer.NextToken()
	if err != nil {
		return err
	}

	var tokenList []Token
	for {
		switch token.Kind {
		case tDirectiveName:
			if len(tokenList) != 0 {
				// TODO: process directive
			}
			fallthrough
		case tDirectiveArgument:
			tokenList = append(tokenList, token)
		case tRaw:
			state.Emitter.WriteIM(token.Data)
		case tEOF:
			if len(tokenList) != 0 {
				// TODO: process directive
			}

			return nil
		}

		token, err = lexer.NextToken()
		if err != nil {
			return err
		}
	}
}

func writeOutput(state *State) error {
	// TODO: implement
	return nil
}

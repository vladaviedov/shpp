/**
 * @file process.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version 0.9.0
 * @date 2026
 * @license GPLv3.0
 * @brief Input file processing logic.
 */
package main

import (
	"fmt"
	"io"
)

type FilepathResolver uint64

const (
	frNormal FilepathResolver = iota
	frRoot
	frParent
)

type State struct {
	PageURL  string
	Assets   []Asset
	Emitter  *Emitter
	ArgCount uint64
	ArgData  []string
	Vars     map[string]string

	FileDir       string
	ParentFileDir string
	PathResolver  FilepathResolver

	IsChild    bool
	UrlChanged bool
	IsTemplate bool
	Ignored    bool
}

func processFile(inStream io.Reader, outStream io.Writer, defaultName string, workingDir string) error {
	state := &State{
		PageURL:      defaultName + ".html",
		FileDir:      workingDir,
		ArgCount:     0,
		ArgData:      nil,
		Emitter:      newEmitter(outStream),
		Vars:         make(map[string]string),
		PathResolver: frNormal,
	}
	defer state.Emitter.destroy()

	err := processFileIM(inStream, state)
	if err != nil {
		return err
	}

	return writeOutput(state)
}

func processFileIM(stream io.Reader, state *State) error {
	lexer, err := newLexer(stream)
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
	token, err := lexer.nextToken()
	if err != nil {
		return err
	}

	var tokenList []Token
	for {
		switch token.Kind {
		case tDirectiveName:
			if len(tokenList) != 0 {
				err = evaluateDirective(tokenList, state, true)
				if err != nil {
					return err
				}
				tokenList = []Token{}
			}
			fallthrough
		case tDirectiveArgument:
			tokenList = append(tokenList, token)
		case tPreambleEnd:
			if len(tokenList) != 0 {
				err = evaluateDirective(tokenList, state, true)
				if err != nil {
					return err
				}
				tokenList = []Token{}
			}

			return nil
		}

		// Short circuit if ignore flag is raised
		if state.Ignored {
			return nil
		}

		token, err = lexer.nextToken()
		if err != nil {
			return err
		}
	}
}

func processBody(lexer *Lexer, state *State) error {
	token, err := lexer.nextToken()
	if err != nil {
		return err
	}

	var tokenList []Token
	for {
		switch token.Kind {
		case tDirectiveName:
			if len(tokenList) != 0 {
				err = evaluateDirective(tokenList, state, false)
				if err != nil {
					return err
				}
				tokenList = []Token{}
			}
			fallthrough
		case tDirectiveArgument:
			tokenList = append(tokenList, token)
		case tRaw:
			if len(tokenList) != 0 {
				err = evaluateDirective(tokenList, state, false)
				if err != nil {
					return err
				}
				tokenList = []Token{}
			}
			state.Emitter.writeIM(token.Data)
		case tEOF:
			if len(tokenList) != 0 {
				err = evaluateDirective(tokenList, state, false)
				if err != nil {
					return err
				}
				tokenList = []Token{}
			}

			return nil
		}

		token, err = lexer.nextToken()
		if err != nil {
			return err
		}
	}
}

func writeOutput(state *State) error {
	if state.Ignored {
		// Shac needs a phony document
		if opts.ShacInput {
			return state.Emitter.writeOut("@ignore\n")
		}

		return nil
	}

	if opts.ShacInput {
		err := writeShacPreamble(state)
		if err != nil {
			return err
		}
	}

	// Write doctype
	err := state.Emitter.writeOut("<!DOCTYPE html>")
	if err != nil {
		return err
	}

	// Locate start of head
	err = findHead(state)
	if err != nil {
		return err
	}

	// Insert stuff into head
	err = updateHead(state)
	if err != nil {
		return err
	}

	// Flush rest of file down
	return state.Emitter.flushIM()
}

func writeShacPreamble(state *State) error {
	err := state.Emitter.writeOut("@page " + state.PageURL + "\n")
	if err != nil {
		return err
	}

	for _, asset := range state.Assets {
		err = state.Emitter.writeOut("@asset " + asset.SourcePath + "\n")
		if err != nil {
			return err
		}
	}

	return state.Emitter.writeOut("@html\n")
}

func findHead(state *State) error {
	imReader := state.Emitter.getIMReader()

	// Find head
	target := "<head>"
	charsMatched := 0
	for {
		ch, err := imReader.ReadByte()
		if err == io.EOF {
			return fmt.Errorf("syntax error: no head")
		}
		if err != nil {
			return err
		}

		// Can send straight to output - no need to hold
		state.Emitter.writeOut(string(ch))

		// To lowercase
		if ch >= 'A' && ch <= 'Z' {
			ch += 0x20
		}

		if ch == target[charsMatched] {
			charsMatched++
		}
		if charsMatched == len(target) {
			// Found head
			break
		}
	}

	return nil
}

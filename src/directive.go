/**
 * @file directive.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief Directive evaluation.
 */
package main

import (
	"fmt"
	"os"
	"strconv"
)

type DirectiveKind uint64

const (
	dNop DirectiveKind = iota
	dIgnore
	dStyle
	dScript
	dUrl
	dInclude
	dPandoc
	dRun
	dManage
	dDefine
	dTemplate
	dDerive
)

type DirectiveDescription struct {
	Kind         DirectiveKind
	RequiredArgs int
	OptionalArgs int
}

const ddUnlimited = -1

var directiveDict = map[string]DirectiveDescription{
	// General
	"nop":    {Kind: dNop, RequiredArgs: 0, OptionalArgs: ddUnlimited},
	"style":  {Kind: dStyle, RequiredArgs: 1, OptionalArgs: 0},
	"script": {Kind: dScript, RequiredArgs: 1, OptionalArgs: 1},
	// Preamble
	"ignore":   {Kind: dIgnore, RequiredArgs: 0, OptionalArgs: 0},
	"url":      {Kind: dUrl, RequiredArgs: 1, OptionalArgs: 0},
	"template": {Kind: dTemplate, RequiredArgs: 1, OptionalArgs: 0},
	// Body
	"include": {Kind: dInclude, RequiredArgs: 1, OptionalArgs: 0},
	"pandoc":  {Kind: dPandoc, RequiredArgs: 1, OptionalArgs: 1},
	"run":     {Kind: dRun, RequiredArgs: 1, OptionalArgs: ddUnlimited},
	"manage":  {Kind: dManage, RequiredArgs: 0, OptionalArgs: 1},
	"derive":  {Kind: dDerive, RequiredArgs: 1, OptionalArgs: ddUnlimited},
}

var inverseMap = map[DirectiveKind]string{
	// General
	dNop:    "nop",
	dStyle:  "style",
	dScript: "script",
	// Preamble
	dIgnore:   "ignore",
	dUrl:      "url",
	dTemplate: "template",
	// Body
	dInclude: "include",
	dPandoc:  "pandoc",
	dRun:     "run",
	dManage:  "manage",
	dDerive:  "derive",
}

type Directive struct {
	Kind      DirectiveKind
	Arguments []string
}

func EvaluateDirective(tokenList []Token, state *State, isPreamble bool) error {
	directive, err := parseDirective(tokenList)
	if err != nil {
		return err
	}

	if isPreamble {
		return evalInPreamble(directive, state)
	} else {
		return evalInBody(directive, state)
	}
}

func parseDirective(tokenList []Token) (*Directive, error) {
	directive := tokenList[0]
	arguments := tokenList[1:]

	// Dictionary lookup
	info, valid := directiveDict[directive.Data]
	if !valid {
		return nil, fmt.Errorf("syntax error: invalid directive '@%s'", directive)
	}

	// Argument count checker
	minArgs := info.RequiredArgs
	maxArgs := info.RequiredArgs + info.OptionalArgs
	if len(arguments) < minArgs {
		return nil, fmt.Errorf("syntax error: too few arguments for '@%s' (%d min)", directive, minArgs)
	}
	if len(arguments) > maxArgs && info.OptionalArgs != ddUnlimited {
		return nil, fmt.Errorf("syntax error: too many arguments for '@%s' (%d max)", directive, maxArgs)
	}

	// Subsitute arguments
	processedArguments := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		subbed, err := substituteArgument(argument.Data)
		if err != nil {
			return nil, err
		}

		processedArguments = append(processedArguments, subbed)
	}

	return &Directive{
		Kind:      info.Kind,
		Arguments: processedArguments,
	}, nil
}

func substituteArgument(argument string) (string, error) {
	// TODO: implement
	return "", nil
}

func evalInPreamble(directive *Directive, state *State) error {
	switch directive.Kind {
	case dNop:
	case dStyle:
		// TODO: implement asset stuff
	case dScript:
		// TODO: implement asset stuff
	case dIgnore:
		state.Ignored = true
	case dUrl:
		// Warn on multiple url changes
		if state.UrlChanged {
			fmt.Fprintf(os.Stderr, "warning: @url directive found multiple times\n")
		}

		state.PageURL = directive.Arguments[0]
		state.UrlChanged = true
	case dTemplate:
		// Error on multiple template definitions
		if state.IsTemplate {
			return fmt.Errorf("syntax error: multiple @template directives found")
		}

		// Parse expected arg count
		expected, err := strconv.ParseUint(directive.Arguments[0], 10, 64)
		if err != nil {
			return fmt.Errorf("syntax error: @template argument must be a number")
		}

		// Check if argument count is valid
		if state.ArgCount != expected {
			return fmt.Errorf("syntax error: @template derived with incorrect argument count")
		}
	default:
		return fmt.Errorf("syntax error: '@%s' not allowed in preamble", inverseMap[directive.Kind])
	}

	return nil
}

func evalInBody(directive *Directive, state *State) error {
	switch directive.Kind {
	case dNop:
	case dStyle:
		// TODO: implement asset stuff
	case dScript:
		// TODO: implement asset stuff
	case dInclude:
	case dPandoc:
	case dRun:
	case dManage:
	case dDerive:
	default:
		return fmt.Errorf("syntax error: '@%s' not allowed in body", inverseMap[directive.Kind])
	}

	return nil
}

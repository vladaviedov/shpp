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
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

type DirectiveKind uint64

const (
	dNop DirectiveKind = iota
	dPathfrom
	dStyle
	dScript
	dIgnore
	dUrl
	dTemplate
	dInclude
	dDerive
	dPandoc
	dRun
	dManage
	dInterp
)

type DirectiveDescription struct {
	Kind         DirectiveKind
	RequiredArgs int
	OptionalArgs int
}

const ddUnlimited = -1

var directiveDict = map[string]DirectiveDescription{
	// General
	"nop":      {Kind: dNop, RequiredArgs: 0, OptionalArgs: ddUnlimited},
	"pathfrom": {Kind: dPathfrom, RequiredArgs: 1, OptionalArgs: 0},
	// Preamble
	"style":    {Kind: dStyle, RequiredArgs: 1, OptionalArgs: 0},
	"script":   {Kind: dScript, RequiredArgs: 1, OptionalArgs: 1},
	"ignore":   {Kind: dIgnore, RequiredArgs: 0, OptionalArgs: 0},
	"url":      {Kind: dUrl, RequiredArgs: 1, OptionalArgs: 0},
	"template": {Kind: dTemplate, RequiredArgs: 1, OptionalArgs: 0},
	// Body
	"include": {Kind: dInclude, RequiredArgs: 1, OptionalArgs: 0},
	"derive":  {Kind: dDerive, RequiredArgs: 1, OptionalArgs: ddUnlimited},
	"pandoc":  {Kind: dPandoc, RequiredArgs: 1, OptionalArgs: 1},
	"run":     {Kind: dRun, RequiredArgs: 1, OptionalArgs: ddUnlimited},
	"manage":  {Kind: dManage, RequiredArgs: 1, OptionalArgs: 0},
	"interp":  {Kind: dInterp, RequiredArgs: 0, OptionalArgs: ddUnlimited},
}

var inverseMap = map[DirectiveKind]string{
	// General
	dNop: "nop",
	// Preamble
	dStyle:    "style",
	dScript:   "script",
	dIgnore:   "ignore",
	dUrl:      "url",
	dTemplate: "template",
	// Body
	dInclude: "include",
	dDerive:  "derive",
	dPandoc:  "pandoc",
	dRun:     "run",
	dManage:  "manage",
	dInterp:  "interp",
}

var scriptModifierMap = map[string]AssetKind{
	"defer": aScriptDefer,
	"async": aScriptAsync,
	"":      aScript,
}

var pathResolverMap = map[string]FilepathResolver{
	"normal": frNormal,
	"root":   frRoot,
	"parent": frParent,
}

type Directive struct {
	Kind      DirectiveKind
	Arguments []string
}

func evaluateDirective(tokenList []Token, state *State, isPreamble bool) error {
	directive, err := parseDirective(tokenList, state)
	if err != nil {
		return err
	}

	if isPreamble {
		return evalInPreamble(directive, state)
	} else {
		return evalInBody(directive, state)
	}
}

func parseDirective(tokenList []Token, state *State) (*Directive, error) {
	directive := tokenList[0]
	arguments := tokenList[1:]

	// Dictionary lookup
	info, valid := directiveDict[directive.Data]
	if !valid {
		return nil, fmt.Errorf("syntax error: invalid directive '@%s'", directive.Data)
	}

	// Argument count checker
	minArgs := info.RequiredArgs
	maxArgs := info.RequiredArgs + info.OptionalArgs
	if len(arguments) < minArgs {
		return nil, fmt.Errorf("syntax error: too few arguments for '@%s' (%d min)", directive.Data, minArgs)
	}
	if len(arguments) > maxArgs && info.OptionalArgs != ddUnlimited {
		return nil, fmt.Errorf("syntax error: too many arguments for '@%s' (%d max)", directive.Data, maxArgs)
	}

	// Subsitute arguments
	processedArguments := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		subbed, err := substituteArgument(argument.Data, state)
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

func substituteArgument(argument string, state *State) (string, error) {
	reader := strings.NewReader(argument)
	var parsed strings.Builder

	for {
		c, _, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch c {
		case '\\':
			next, _, err := reader.ReadRune()
			if err != nil {
				return "", err
			}

			if next == '{' || next == '}' {
				parsed.WriteRune(next)
			} else {
				parsed.WriteRune(c)
				reader.UnreadRune()
			}
		case '{':
			var ident strings.Builder
			for {
				c, _, err = reader.ReadRune()
				if err != nil {
					return "", err
				}

				if c == '}' {
					break
				}

				ident.WriteRune(c)
			}

			subVal, err := getVar(state, ident.String())
			if err != nil {
				return "", err
			}

			parsed.WriteString(subVal)
		default:
			parsed.WriteRune(c)
		}

	}

	return parsed.String(), nil
}

func evalInPreamble(directive *Directive, state *State) error {
	switch directive.Kind {
	case dNop:
	case dPathfrom:
		resolver, valid := pathResolverMap[directive.Arguments[0]]
		if !valid {
			return fmt.Errorf("syntax error: unknown filepath resolver: '%s'", directive.Arguments[0])
		}

		state.PathResolver = resolver
	case dStyle:
		asset := createAsset(aStylesheet, state.FileDir, directive.Arguments[0])
		state.Assets = append(state.Assets, asset)
	case dScript:
		modifier := ""
		if len(directive.Arguments) == 2 {
			modifier = directive.Arguments[1]
		}

		scriptKind, valid := scriptModifierMap[modifier]
		if !valid {
			return fmt.Errorf("syntax error: unknown @script modifier '%s'", modifier)
		}

		asset := createAsset(scriptKind, state.FileDir, directive.Arguments[0])
		state.Assets = append(state.Assets, asset)
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
		return state.Emitter.writeIM("<!-- NOP -->")
	case dPathfrom:
		resolver, valid := pathResolverMap[directive.Arguments[0]]
		if !valid {
			return fmt.Errorf("syntax error: unknown filepath resolver: '%s'", directive.Arguments[0])
		}

		state.PathResolver = resolver
	case dInclude:
		fallthrough
	case dDerive:
		return includeCommon(directive.Arguments[0], state, directive.Arguments[1:])
	case dPandoc:
		format := ""
		if len(directive.Arguments) == 2 {
			format = directive.Arguments[1]
		}

		return includePandoc(directive.Arguments[0], format, state)
	case dRun:
		if !opts.ShellEscape {
			fmt.Fprintf(os.Stderr, "warning: @run ignored (shell escape is disabled)\n")
		}

		result, err := runSystemCommand(strings.Join(directive.Arguments, " "))
		if err != nil {
			return err
		}

		return state.Emitter.writeIM(result)
	case dManage:
		if !opts.ShacInput {
			fmt.Fprintf(os.Stderr, "warning: @manage ignored (shac integration not enabled)\n")

			// Replace with just the argument
			return state.Emitter.writeIM(directive.Arguments[0])
		}

		state.Assets = append(state.Assets, createAsset(aBinary, resolveFileDir(state), directive.Arguments[0]))
		return state.Emitter.writeIM("@" + strconv.Itoa(len(state.Assets)-1) + "@")
	case dInterp:
		return state.Emitter.writeIM(strings.Join(directive.Arguments, " "))
	default:
		return fmt.Errorf("syntax error: '@%s' not allowed in body", inverseMap[directive.Kind])
	}

	return nil
}

func includeCommon(filepath string, state *State, withArgs []string) error {
	absPath := convertPath(filepath, resolveFileDir(state))
	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer file.Close()

	// Create new state
	childState := &State{
		Assets:        state.Assets,
		PageURL:       state.PageURL,
		FileDir:       path.Dir(absPath),
		ArgCount:      uint64(len(withArgs)),
		ArgData:       withArgs,
		Vars:          state.Vars,
		Emitter:       state.Emitter,
		IsChild:       true,
		ParentFileDir: state.FileDir,
		PathResolver:  frNormal,
	}

	// Add inclusion marker
	if opts.Marker {
		marker := fmt.Sprintf("<!-- START %s -->", filepath)
		state.Emitter.writeIM(marker)
	}

	// Process file
	err = processFileIM(file, childState)
	if err != nil {
		return err
	}

	// Add inclusion marker
	if opts.Marker {
		marker := fmt.Sprintf("<!-- END %s -->", filepath)
		state.Emitter.writeIM(marker)
	}

	// Copy assets from child
	state.Assets = childState.Assets
	return nil
}

func includePandoc(filepath string, modifier string, state *State) error {
	absPath := convertPath(filepath, resolveFileDir(state))
	inputFormat := modifier

	if !opts.ShellEscape {
		fmt.Fprintf(os.Stderr, "warning: @pandoc ignored (shell escape is disabled)\n")
		return nil
	}

	// Run pandoc logic
	pandocOut, err := pandocConvert(absPath, inputFormat)
	if err != nil {
		return err
	}
	defer os.Remove(pandocOut.Name())

	// Create new state
	childState := &State{
		Assets:        state.Assets,
		PageURL:       state.PageURL,
		FileDir:       path.Dir(absPath),
		Vars:          state.Vars,
		Emitter:       state.Emitter,
		IsChild:       true,
		ParentFileDir: state.FileDir,
		PathResolver:  frNormal,
	}

	// Add inclusion marker
	if opts.Marker {
		marker := fmt.Sprintf("<!-- START %s -->", filepath)
		state.Emitter.writeIM(marker)
	}

	// Process file
	err = processFileIM(pandocOut, childState)
	if err != nil {
		return err
	}

	// Add inclusion marker
	if opts.Marker {
		marker := fmt.Sprintf("<!-- END %s -->", filepath)
		state.Emitter.writeIM(marker)
	}

	// Copy assets from child
	state.Assets = childState.Assets
	return nil
}

func resolveFileDir(state *State) string {
	switch state.PathResolver {
	case frNormal:
		return state.FileDir
	case frRoot:
		return opts.Root
	case frParent:
		return state.ParentFileDir
	}

	panic("invalil file resolver state")
}

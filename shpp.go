package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var opts struct {
	Help      bool `long:"help" short:"h"`
	Version   bool `long:"version" short:"v"`
	Stdin     bool `long:"stdin" short:"x"`
	Marker    bool `long:"marker" short:"m"`
	ShacInput bool `long:"shac" short:"c"`

	Output string `long:"output" short:"o"`
}

type AssetKind uint64

const (
	aStylesheet AssetKind = iota
	aScript
	aBinary
)

type DirectiveKind uint64

const (
	dNop DirectiveKind = iota
	dHtml
	dStyle
	dScript
	dUrl
	dInclude
	dManage
)

type Asset struct {
	Kind       AssetKind
	SourcePath string
}

type Directive struct {
	Kind DirectiveKind
	Args []string
}

type State struct {
	Assets  []Asset
	PageURL string
	FileDir string
}

type DirectiveDescription struct {
	Kind         DirectiveKind
	RequiredArgs uint64
	OptionalArgs uint64
}

var directiveDict = map[string]DirectiveDescription{
	"@style":   {Kind: dStyle, RequiredArgs: 1, OptionalArgs: 0},
	"@script":  {Kind: dScript, RequiredArgs: 1, OptionalArgs: 0},
	"@url":     {Kind: dUrl, RequiredArgs: 1, OptionalArgs: 0},
	"@include": {Kind: dInclude, RequiredArgs: 1, OptionalArgs: 0},
	"@manage":  {Kind: dManage, RequiredArgs: 0, OptionalArgs: 1},
}

// Populated by build system
var Version string = "0.3.0"

func main() {
	parser := flags.NewParser(&opts, flags.Default^flags.HelpFlag^flags.PrintErrors)
	args, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse arguments: %s\n", err.Error())
		os.Exit(2)
	}

	if opts.Help {
		usage(os.Stdout)
		os.Exit(0)
	}
	if opts.Version {
		version()
		os.Exit(0)
	}

	var outStream *os.File = nil
	if opts.Output == "" {
		outStream = os.Stdout
	} else {
		outStream, err = os.Create(opts.Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open output file: %s\n", err.Error())
			os.Exit(1)
		}
	}
	defer outStream.Close()

	if (!opts.Stdin && len(args) != 1) || (opts.Stdin && len(args) != 0) {
		usage(os.Stderr)
		os.Exit(2)
	}

	var inStream *os.File
	var inWorkingDir string
	if opts.Stdin {
		inStream = os.Stdin
		inWorkingDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to fetch current directory: %s\n", err.Error())
			os.Exit(1)
		}
	} else {
		inStream, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open source file: %s\n", err.Error())
			os.Exit(1)
		}

		inWorkingDir, err = filepath.Abs(path.Dir(args[0]))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to determined working directory: %s\n", err.Error())
			os.Exit(1)
		}
	}
	defer inStream.Close()

	// Initial parser state
	var defaultName string
	if opts.Stdin {
		defaultName = "stdin"
	} else {
		defaultName, _ = strings.CutSuffix(inStream.Name(), ".shpp")
	}
	state := &State{
		PageURL: defaultName + ".html",
		FileDir: inWorkingDir,
	}

	wrappedDocument, err := compile(inStream, state, nil)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	document := wrappedDocument.FirstChild

	// Place any necessary tags into the head
	err = updateHead(document, state)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	// shac preamble
	if opts.ShacInput {
		writeShacPreamble(outStream, state)
	}

	// Unwrap from the phony and write to output
	fmt.Fprintf(outStream, "<!DOCTYPE html>")
	html.Render(outStream, document)

	// Add end-of-file newline
	outStream.WriteString("\n")
}

func usage(toFile *os.File) {
	fmt.Fprintf(toFile, "usage: %s [options] <source>\n", os.Args[0])
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-x, --stdin", "Read input file from stdin (source should be left empty)")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-h, --help", "Show usage information")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-v, --version", "Show program version")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-m, --marker", "Insert inclusion markers")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-c, --shac", "Generate 'shac' input file")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-o, --output <file>", "Set output file")
}

func version() {
	fmt.Printf("shpp version %s\n", Version)
}

func compile(file *os.File, state *State, htmlContext *html.Node) (*html.Node, error) {
	reader := bufio.NewReader(file)

	// Preamble
	err := readPreamble(reader, state)
	if err != nil {
		return nil, err
	}

	tagList, err := html.ParseFragment(reader, htmlContext)
	if err != nil {
		msg := fmt.Sprintf("failed to parse HTML document: %s\n", err.Error())
		return nil, errors.New(msg)
	}

	// Wrap the parsed tag into a phony tag
	// This simplifies travesal and ensures that a parent always exists
	phony := &html.Node{}

	// The node will mimic the context node if it's not null
	if htmlContext != nil {
		phony.Type = htmlContext.Type
		phony.Data = htmlContext.Data
		phony.DataAtom = htmlContext.DataAtom
	}

	for _, tag := range tagList {
		phony.AppendChild(tag)
	}

	for node := range phony.Descendants() {
		err = processNode(node, state)
		if err != nil {
			return nil, err
		}
	}

	return phony, nil
}

func readPreamble(reader *bufio.Reader, state *State) error {
	urlChanged := false

	for {
		nextChar, err := reader.Peek(1)
		if err != nil {
			return err
		}

		// Reached HTML section
		if nextChar[0] == '<' {
			break
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		dr, err := parseDirective(line)
		if err != nil {
			return err
		}

		switch dr.Kind {
		case dInclude:
			return errors.New("syntax error: @include not allowed in preamble\n")
		case dManage:
			return errors.New("syntax error: @manage not allowed in preamble\n")
		case dStyle:
			asset := createAsset(aStylesheet, state.FileDir, dr.Args[0])
			state.Assets = append(state.Assets, asset)
		case dScript:
			asset := createAsset(aScript, state.FileDir, dr.Args[0])
			state.Assets = append(state.Assets, asset)
		case dUrl:
			if urlChanged {
				fmt.Fprintf(os.Stderr, "warning: multiple @url directives found\n")
			}

			state.PageURL = dr.Args[0]
			urlChanged = true
		}
	}

	return nil
}

func parseDirective(input string) (*Directive, error) {
	trimmed := strings.Trim(input, " \t\n")

	// Empty lines and comment lines
	if len(trimmed) == 0 || strings.HasPrefix(trimmed, "//") {
		return &Directive{
			Kind: dNop,
			Args: nil,
		}, nil
	}

	// All directive must start with a @
	if trimmed[0] != '@' {
		return &Directive{
			Kind: dHtml,
			Args: nil,
		}, nil
	}

	parts := strings.Split(trimmed, " ")
	info, valid := directiveDict[parts[0]]

	// TODO: would be nice to give line information
	if !valid {
		msg := fmt.Sprintf("syntax error: invalid directive '%s'\n", parts[0])
		return nil, errors.New(msg)
	}

	minArgs := info.RequiredArgs
	maxArgs := minArgs + info.OptionalArgs
	args := uint64(len(parts)) - 1
	if args < minArgs || args > maxArgs {
		var msg string
		if minArgs != maxArgs {
			msg = fmt.Sprintf("syntax error: '%s' requires %d-%d arguments\n", parts[0], minArgs, maxArgs)
		} else {
			msg = fmt.Sprintf("syntax error: '%s' requires %d arguments\n", parts[0], minArgs)
		}

		return nil, errors.New(msg)
	}

	return &Directive{
		Kind: info.Kind,
		Args: parts[1:],
	}, nil
}

func createAsset(kind AssetKind, basePath string, path string) Asset {
	path, err := filepath.Abs(filepath.Join(basePath, path))
	if err != nil {
		panic(err)
	}

	return Asset{
		Kind:       kind,
		SourcePath: path,
	}
}

func processNode(node *html.Node, state *State) error {
	// Only text nodes should be handled here
	if node.Type != html.TextNode {
		return nil
	}

	lines := strings.Split(node.Data, "\n")
	unusedText := new(strings.Builder)

	for i, line := range lines {
		dr, err := parseDirective(line)
		if err != nil {
			return err
		}

		switch dr.Kind {
		case dInclude:
			absPath := convertPath(dr.Args[0], state.FileDir)
			file, err := os.Open(absPath)
			if err != nil {
				return errors.New("failed to open source file: %s\n")
			}
			defer file.Close()

			oldDir := state.FileDir
			state.FileDir = path.Dir(absPath)
			wrappedTags, err := compile(file, state, node.Parent)
			state.FileDir = oldDir
			if err != nil {
				return err
			}

			// Any text before the include become a text node
			if unusedText.Len() != 0 {
				textNode := &html.Node{
					Type: html.TextNode,
					Data: unusedText.String(),
				}

				node.Parent.InsertBefore(textNode, node)
				unusedText.Reset()
			}

			// Add inclusion marker
			if opts.Marker {
				marker := &html.Node{
					Type: html.CommentNode,
					Data: fmt.Sprintf("START %s", dr.Args[0]),
				}
				node.Parent.InsertBefore(marker, node)
			}

			// Now add included nodes
			tag := wrappedTags.FirstChild
			for tag != nil {
				// Appease the HTML parser
				wrappedTags.RemoveChild(tag)

				// Push all the new tags before the original one
				node.Parent.InsertBefore(tag, node)

				// Update tag reference
				tag = wrappedTags.FirstChild
			}

			// Add inclusion marker
			if opts.Marker {
				marker := &html.Node{
					Type: html.CommentNode,
					Data: fmt.Sprintf("END %s", dr.Args[0]),
				}
				node.Parent.InsertBefore(marker, node)
			}
		case dManage:
			if !opts.ShacInput {
				fmt.Fprintf(os.Stderr, "warning: @manage ignored (--shac not enabled)\n")
				break
			}

			target := node.NextSibling
			if target == nil || target.Type != html.ElementNode {
				return errors.New("syntax error: @manage must be followed by an element\n")
			}

			// Use 'src' by default
			attrName := "src"
			if len(dr.Args) == 1 {
				attrName = dr.Args[0]
			}

			// Find attribute
			index := slices.IndexFunc(target.Attr, func(a html.Attribute) bool {
				return a.Key == attrName
			})
			if index < 0 {
				msg := fmt.Sprintf("syntax error: @manage target is missing '%s' attribute\n", attrName)
				return errors.New(msg)
			}
			attr := &target.Attr[index].Val

			// Record asset and replace with shac placeholder
			state.Assets = append(state.Assets, createAsset(aBinary, state.FileDir, *attr))
			*attr = "@" + strconv.Itoa(len(state.Assets)-1) + "@"
		case dStyle:
			return errors.New("syntax error: @style must be placed in the preamble\n")
		case dScript:
			return errors.New("syntax error: @script must be placed in the preamble\n")
		case dNop:
			fallthrough
		case dHtml:
			// Store used but uncommited text (for splitting the node)
			unusedText.WriteString(line)
			if i != len(lines)-1 {
				unusedText.WriteString("\n")
			}
		}
	}

	// Remaining text stays in this node
	node.Data = unusedText.String()
	return nil
}

func convertPath(path string, fileDir string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(fileDir, path)
}

func updateHead(document *html.Node, state *State) error {
	head := findHead(document)

	for i, asset := range state.Assets {
		headAsset, err := generateHeadAsset(&asset, i)
		if err != nil {
			return err
		}

		if headAsset != nil {
			head.AppendChild(headAsset)
		}
	}

	// On shac files, we want to add a base tag
	if !opts.ShacInput {
		return nil
	}

	// Done, if base tag already exists
	for child := range head.ChildNodes() {
		if child.DataAtom == atom.Base {
			return nil
		}
	}

	// Base should be placed before any links
	head.InsertBefore(&html.Node{
		Type:     html.ElementNode,
		Data:     "base",
		DataAtom: atom.Base,
		Attr: []html.Attribute{
			{
				Key: "href",
				Val: "@$@/",
			},
		},
	}, head.FirstChild)

	return nil
}

func findHead(document *html.Node) *html.Node {
	for node := range document.ChildNodes() {
		if node.DataAtom == atom.Head {
			return node
		}
	}

	panic("generated document is missing the 'head' tag")
}

func generateHeadAsset(asset *Asset, id int) (*html.Node, error) {
	switch asset.Kind {
	case aStylesheet:
		return generateStyleTag(asset, id)
	case aScript:
		return generateScriptTag(asset, id)
	default:
		// No head tag required for this asset
		return nil, nil
	}
}

func generateStyleTag(asset *Asset, id int) (*html.Node, error) {
	if opts.ShacInput {
		// <link rel="stylesheet" href="@id@">
		return &html.Node{
			Type:     html.ElementNode,
			Data:     "link",
			DataAtom: atom.Link,
			Attr: []html.Attribute{
				{
					Key: "rel",
					Val: "stylesheet",
				},
				{
					Key: "href",
					Val: "@" + strconv.Itoa(id) + "@",
				},
			},
		}, nil
	} else {
		// <style>contents</style>
		fileContent, err := os.ReadFile(asset.SourcePath)
		if err != nil {
			return nil, err
		}

		// Text inside of <style>
		contentNode := &html.Node{
			Type: html.TextNode,
			Data: string(fileContent),
		}
		// Actual <style> tag
		tag := &html.Node{
			Type:     html.ElementNode,
			Data:     "style",
			DataAtom: atom.Style,
		}

		tag.AppendChild(contentNode)
		return tag, nil
	}
}

func generateScriptTag(asset *Asset, id int) (*html.Node, error) {
	if opts.ShacInput {
		// <script src="@id@">
		return &html.Node{
			Type:     html.ElementNode,
			Data:     "script",
			DataAtom: atom.Script,
			Attr: []html.Attribute{
				{
					Key: "src",
					Val: "@" + strconv.Itoa(id) + "@",
				},
			},
		}, nil
	} else {
		// <script>contents</script>
		fileContent, err := os.ReadFile(asset.SourcePath)
		if err != nil {
			return nil, err
		}

		// Text inside of <script>
		contentNode := &html.Node{
			Type: html.TextNode,
			Data: string(fileContent),
		}
		// Actual <script> tag
		tag := &html.Node{
			Type:     html.ElementNode,
			Data:     "script",
			DataAtom: atom.Script,
		}

		tag.AppendChild(contentNode)
		return tag, nil
	}
}

func writeShacPreamble(file *os.File, state *State) {
	fmt.Fprintf(file, "@page %s\n", state.PageURL)
	for _, asset := range state.Assets {
		fmt.Fprintf(file, "@asset %s\n", asset.SourcePath)
	}
	fmt.Fprintf(file, "@html\n")
}

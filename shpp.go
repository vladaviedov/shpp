package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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
	dInclude
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
	"@include": {Kind: dInclude, RequiredArgs: 1, OptionalArgs: 0},
}

// Populated by build system
var Version string = "0.1.0"

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

	wrappedDocument, finalState, err := compile(inStream, inWorkingDir, nil)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	document := wrappedDocument.FirstChild

	// Place stylesheets and scripts into the head
	err = placeHeadAssets(document, finalState)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	// TODO: write asset metadata

	// Unwrap from the phony and write to output
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

func compile(file *os.File, fileDir string, htmlContext *html.Node) (*html.Node, *State, error) {
	defaultName, _ := strings.CutSuffix(file.Name(), ".in")
	state := &State{
		PageURL: defaultName,
		FileDir: fileDir,
	}

	// Preamble
	err := readPreamble(file, state)
	if err != nil {
		return nil, nil, err
	}

	reader := bufio.NewReader(file)
	tagList, err := html.ParseFragment(reader, htmlContext)
	if err != nil {
		msg := fmt.Sprintf("failed to parse HTML document: %s\n", err.Error())
		return nil, nil, errors.New(msg)
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
			return nil, nil, err
		}
	}

	return phony, state, nil
}

func readPreamble(file *os.File, state *State) error {
	reader := bufio.NewReader(file)
	consumed := int64(0)

	kind := dNop
	for kind != dHtml {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		dr, err := parseDirective(line)
		if err != nil {
			return err
		}

		kind = dr.Kind
		switch kind {
		case dInclude:
			return errors.New("syntax error: @include not allowed in preamble\n")
		case dStyle:
			fallthrough
		case dScript:
			asset := createAsset(dr, state.FileDir)
			state.Assets = append(state.Assets, asset)
			fallthrough
		case dNop:
			consumed += int64(len(line))
		}
	}

	// Rewind file to "place" back the html data
	file.Seek(consumed, io.SeekStart)
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

func createAsset(dr *Directive, basePath string) Asset {
	var asset Asset

	switch dr.Kind {
	case dStyle:
		asset.Kind = aStylesheet
	case dScript:
		asset.Kind = aScript
	default:
		panic("Unable to create an asset from directive")
	}

	path, err := filepath.Abs(filepath.Join(basePath, dr.Args[0]))
	if err != nil {
		panic(err)
	}

	asset.SourcePath = path
	return asset
}

func processNode(node *html.Node, state *State) error {
	// Only text nodes should be handled here
	if node.Type != html.TextNode {
		return nil
	}

	lines := strings.Split(node.Data, "\n")
	unusedText := new(strings.Builder)

	for _, line := range lines {
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

			// TODO: need to merge state assets
			wrappedTags, _, err := compile(file, path.Dir(absPath), node.Parent)
			if err != nil {
				return err
			}

			// Any text before the include become a text node
			if unusedText.Len() != 0 {
				textNode := &html.Node{
					Type: html.ElementNode,
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
		case dStyle:
			return errors.New("syntax error: @style must be placed in the preamble\n")
		case dScript:
			return errors.New("syntax error: @script must be placed in the preamble\n")
		case dNop:
			fallthrough
		case dHtml:
			// Store used but uncommited text (for splitting the node)
			unusedText.WriteString(strings.Trim(line, " \t\n"))
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

func placeHeadAssets(document *html.Node, state *State) error {
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

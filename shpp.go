package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	Help    bool `long:"help" short:"h"`
	Version bool `long:"version" short:"v"`
	Stdin   bool `long:"stdin" short:"x"`
	Marker  bool `long:"marker" short:"m"`

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
		inWorkingDir = path.Dir(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open source file: %s\n", err.Error())
			os.Exit(1)
		}
	}
	defer inStream.Close()

	outStream.Write(compile(inStream, inWorkingDir))
}

func usage(toFile *os.File) {
	fmt.Fprintf(toFile, "usage: %s [options] <source>\n", os.Args[0])
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-x, --stdin", "Read input file from stdin (source should be left empty)")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-h, --help", "Show usage information")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-v, --version", "Show program version")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-m, --marker", "Insert inclusion markers")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-o, --output <file>", "Set output file")
}

func version() {
	fmt.Printf("shpp version %s\n", Version)
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

func compile(file *os.File, fileDir string) []byte {
	builder := new(strings.Builder)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		trimmed := strings.Trim(scanner.Text(), " \t")
		if len(trimmed) == 0 {
			continue
		}

		if trimmed[0] == '@' {
			result, err := evalDirective(trimmed, fileDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to compile: %s\n", err.Error())
				break
			}

			// Place start marker
			if opts.Marker {
				fmt.Fprintf(builder, "<!-- START %s -->\n", trimmed)
			}

			builder.Write(result)

			// Place stop marker
			if opts.Marker {
				fmt.Fprintf(builder, "<!-- END %s -->", trimmed)
			}
		} else if trimmed[0] == '\\' && trimmed[1] == '@' {
			builder.Write([]byte(trimmed[1:]))
		} else {
			builder.Write([]byte(trimmed))
		}

		fmt.Fprintln(builder)
	}

	return []byte(builder.String())
}

func evalDirective(directive string, fileDir string) ([]byte, error) {
	parts := strings.Split(directive, " ")
	switch parts[0] {
	case "@include":
		return evalInclude(parts, fileDir)
	case "@style":
		return evalStyle(parts, fileDir)
	case "@script":
		return evalScript(parts, fileDir)
	default:
		msg := fmt.Sprintf("invalid directive: %s", parts[0])
		return nil, errors.New(msg)
	}
}

func evalInclude(argv []string, fileDir string) ([]byte, error) {
	if len(argv) != 2 {
		return nil, errors.New("@include: syntax error: requires a single parameter")
	}

	includePath := path.Join(fileDir, argv[1])
	includeDir := path.Dir(includePath)
	includeFile, err := os.Open(includePath)
	if err != nil {
		msg := fmt.Sprintf("@include: failed to open '%s'", err.Error())
		return nil, errors.New(msg)
	}
	defer includeFile.Close()

	return compile(includeFile, includeDir), nil
}

func evalStyle(argv []string, fileDir string) ([]byte, error) {
	if len(argv) != 2 {
		return nil, errors.New("@style: syntax error: requires a single parameter")
	}

	stylePath := path.Join(fileDir, argv[1])
	data, err := os.ReadFile(stylePath)
	if err != nil {
		msg := fmt.Sprintf("@style: failed to read '%s'", err.Error())
		return nil, errors.New(msg)
	}

	builder := new(strings.Builder)
	fmt.Fprintln(builder, "<style>")
	builder.Write(data)
	fmt.Fprintln(builder, "</style>")

	return []byte(builder.String()), nil
}

func evalScript(argv []string, fileDir string) ([]byte, error) {
	if len(argv) != 2 {
		return nil, errors.New("@script: syntax error: requires a single parameter")
	}

	stylePath := path.Join(fileDir, argv[1])
	data, err := os.ReadFile(stylePath)
	if err != nil {
		msg := fmt.Sprintf("@script: failed to read '%s'", err.Error())
		return nil, errors.New(msg)
	}

	builder := new(strings.Builder)
	fmt.Fprintln(builder, "<script>")
	builder.Write(data)
	fmt.Fprintln(builder, "</script>")

	return []byte(builder.String()), nil
}

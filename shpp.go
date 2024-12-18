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

func convertPath(path string, fileDir string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(fileDir, path)
}

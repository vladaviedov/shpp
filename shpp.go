package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	Help bool `long:"help" short:"h"`
	Version bool `long:"version" short:"v"`
	Stdin bool `long:"stdin" short:"x"`

	Output string `long:"output" short:"o"`
}

// Populated by build system
var Version string = "pre-0.1"

func main() {
	parser := flags.NewParser(&opts, flags.Default ^ flags.HelpFlag ^ flags.PrintErrors)
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
	fmt.Fprintf(toFile, "%-20s - %s\n", "-o, --output <file>", "Set output file")
}

func version() {
	fmt.Printf("shpp version %s\n", Version)
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

			fmt.Fprintf(builder, "<!-- START %s -->\n", trimmed)
			builder.Write(result)
			fmt.Fprintf(builder, "<!-- END %s -->", trimmed)
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
		return nil, errors.New("invalid directive")
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
		msg := fmt.Sprintf("@include: failed to read '%s'", err.Error())
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

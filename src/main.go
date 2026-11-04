/**
 * @file main.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 */
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	// General options
	Help    bool   `long:"help" short:"h"`
	Version bool   `long:"version" short:"v"`
	Stdin   bool   `long:"stdin" short:"x"`
	Marker  bool   `long:"marker" short:"m"`
	Output  string `long:"output" short:"o"`
	Root    string `long:"root" short:"r"`

	// Integration
	ShacInput      bool   `long:"shac" short:"c"`
	ShellEscape    bool   `long:"shell-escape" short:"s"`
	PandocBinary   string `long:"pandoc-binary" default:"pandoc"`
	PandocOptsFile string `long:"pandoc-opts"`

	// Other
	IMBufferSize int `long:"imbuf-size" default:"8192"`
}

// Populated by build system
var Version string = "0.7.0"

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
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to fetch current directory: %s\n", err.Error())
		os.Exit(1)
	}

	if opts.Stdin {
		inStream = os.Stdin
		inWorkingDir = cwd
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

	// Make root directory an absolute path
	if !filepath.IsAbs(opts.Root) {
		opts.Root = filepath.Join(cwd, opts.Root)
	}

	// Initial parser state
	var defaultName string
	if opts.Stdin {
		defaultName = "stdin"
	} else {
		defaultName, _ = strings.CutSuffix(inStream.Name(), ".shpp")
	}

	err = processFile(inStream, outStream, defaultName, inWorkingDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}

func usage(toFile *os.File) {
	fmt.Fprintf(toFile, "usage: %s [options] <source>\n", os.Args[0])
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "== General options ==\n")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-x, --stdin", "Read input file from stdin (source should be left empty)")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-h, --help", "Show usage information")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-v, --version", "Show program version")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-m, --marker", "Insert inclusion markers")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-o, --output <file>", "Set output file")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-r, --root <path>", "Set root directory")
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "== Integrations ==\n")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-c, --shac", "Generate 'shac' input file")
	fmt.Fprintf(toFile, "%-25s - %s\n", "-s, --shell-escape", "Enable shell escape")
	fmt.Fprintf(toFile, "%-25s - %s\n", "--pandoc-binary <file>", "Set pandoc binary path")
	fmt.Fprintf(toFile, "%-25s - %s\n", "--pandoc-opts <file>", "Set pandoc options file")
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "== Other ==\n")
	fmt.Fprintf(toFile, "%-25s - %s\n", "--imbuf-size <n>", "Set max size for in-memory intermediate buffer size")
}

func version() {
	fmt.Printf("shpp version %s\n", Version)
}

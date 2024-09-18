package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	Help bool `long:"help" short:"h"`
	Version bool `long:"version" short:"v"`
	Stdin bool `long:"stdin" short:"x"`

	Output string `long:"output" short:"o"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default ^ flags.HelpFlag)
	args, err := parser.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse arguments: %s", err.Error())
		os.Exit(2)
	}

	if opts.Help {
		usage(os.Stdout)
		os.Exit(0)
	}

	var outStream *os.File = nil
	if opts.Output == "" {
		outStream = os.Stdout
	} else {
		outStream, err = os.Create(opts.Output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open output file: %s", err.Error())
			os.Exit(1)
		}
	}

	if len(args) != 1 || (opts.Stdin && len(args) != 0) {
		usage(os.Stderr)
		os.Exit(2)
	}

	outStream.Close()
}

func usage(toFile *os.File) {
	fmt.Fprintf(toFile, "usage: %s [options] <source>\n", os.Args[0])
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-o, --output <file>", "Set output file")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-x, --stdin", "Read input file from stdin (source should be left empty)")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-h, --help", "Show usage information")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-v, --version", "Show program version")
}

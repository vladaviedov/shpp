package main

import (
	"fmt"
	"io"
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

	var inStream *os.File = nil	
	if opts.Stdin {
		inStream = os.Stdin
	} else {
		inStream, err = os.Open(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open source file: %s\n", err.Error())
		}
	}
	defer inStream.Close()

	data, _ := io.ReadAll(inStream)
	outStream.Write(data)
}

func usage(toFile *os.File) {
	fmt.Fprintf(toFile, "usage: %s [options] <source>\n", os.Args[0])
	fmt.Fprintf(toFile, "\n")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-x, --stdin", "Read input file from stdin (source should be left empty)")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-h, --help", "Show usage information")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-v, --version", "Show program version")
	fmt.Fprintf(toFile, "%-20s - %s\n", "-o, --output <file>", "Set output file")
}

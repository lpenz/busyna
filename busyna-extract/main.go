package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lpenz/busyna/libbusyna"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <format: make> <busyna.rc>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = func() {
		usage()
	}

	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(1)
	}
	format := flag.Arg(0)
	output := flag.Arg(1)
	switch format {
	case "make":
		libbusyna.ExtractMake(output)
	default:
		usage()
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lpenz/busyna/libbusyna"
)

func usage() {
	fmt.Printf("Usage: busyna-deploy <format: dot> <input file> <output file>\n\n")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if flag.NArg() != 3 {
		usage()
		os.Exit(1)
	}
	format := flag.Arg(0)
	db := flag.Arg(1)
	dot := flag.Arg(2)
	switch format {
	case "dot":
		libbusyna.DeployDot(libbusyna.DbRead(db), dot)
	default:
		usage()
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lpenz/busyna/libbusyna"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <format: dot|make|busyna|tup> <busyna.db> <output.mk/output.dot>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = func() {
		usage()
	}

	flag.Parse()

	if flag.NArg() != 3 {
		usage()
		os.Exit(1)
	}
	format := flag.Arg(0)
	db := flag.Arg(1)
	output := flag.Arg(2)
	switch format {
	case "busyna":
		libbusyna.DeployRc(libbusyna.DbRead(db), output)
	case "dot":
		libbusyna.DeployDot(libbusyna.DbRead(db), output)
	case "make":
		libbusyna.DeployMake(libbusyna.DbRead(db), output)
	case "tup":
		libbusyna.DeployTup(libbusyna.DbRead(db), output)
	default:
		usage()
		os.Exit(1)
	}
}

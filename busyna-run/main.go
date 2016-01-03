package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lpenz/busyna/libbusyna"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <busyna.rc> <busyna.db>\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}
	rc := flag.Arg(0)
	db := flag.Arg(1)
	libbusyna.DbWrite(libbusyna.RcRun(libbusyna.RcParse("", libbusyna.ChanFromFile(rc))), db)
}

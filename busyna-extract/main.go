package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/lpenz/busyna/libbusyna"
)

func usage() {
	fmt.Printf("Usage: busyna-extract <format: make> <output file>\n\n")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(1)
	}
	format := flag.Arg(0)
	output := flag.Arg(1)
	switch format {
	case "make":
		mkfile, shfile := libbusyna.ExtractMake(output)
		//defer libbusyna.TmpEnd(mkfile)
		//defer libbusyna.TmpEnd(shfile)
		fmt.Printf("mkfile %s, shfile %s\n", mkfile.Name(), shfile.Name())
		err := exec.Command("cat", shfile.Name()).Run()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("mkfile %s, shfile %s\n", mkfile.Name(), shfile.Name())
		cmd := exec.Command("make", "-B", "-f", mkfile.Name())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err != nil {
			log.Fatal(err)
		}
	default:
		usage()
		os.Exit(1)
	}
}

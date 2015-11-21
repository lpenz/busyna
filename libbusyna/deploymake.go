package libbusyna

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// Create a dot graphviz file with db data read form the provided channel.
func DeployMake(c <-chan CmdData, outputfile string) {
	fd, err := ioutil.TempFile(filepath.Dir(outputfile), "Makefile-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(fd.Name())

	// Header:
	fd.WriteString(fmt.Sprintf("# Automatically generated from %s\n\n", outputfile))
	fd.WriteString(".PHONY: all clean\n\n")
	fd.WriteString("all:\n\n")

	// Rules:
	targets := map[string]bool{}
	for cmddata := range c {
		var first string = ""
		for target := range cmddata.Targets {
			if first == "" {
				fd.WriteString(fmt.Sprintf("%s:", target))
				for dep := range cmddata.Deps {
					fd.WriteString(fmt.Sprintf(" %s", dep))
				}
				fd.WriteString(fmt.Sprintf("\n\t%s\n\n", cmddata.Cmd.Line))
				first = target
			} else {
				fd.WriteString(fmt.Sprintf("%s: %s\n\n", target, first))
			}
			targets[target] = true
			fd.WriteString("\n")
		}
	}

	// Fix "all" target, to depend on all targets:
	fd.WriteString("all:")
	for target := range targets {
		fd.WriteString(fmt.Sprintf(" %s", target))
	}
	fd.WriteString("\n\n")

	// Create the "clean" target:
	fd.WriteString("clean:\n\trm -f")
	for target := range targets {
		fd.WriteString(fmt.Sprintf(" '%s'", target))
	}
	fd.WriteString("\n\n")

	fd.Close()

	if err = os.Rename(fd.Name(), outputfile); err != nil {
		log.Fatal(err)
	}
}

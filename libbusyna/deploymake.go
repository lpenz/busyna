package libbusyna

import (
	"fmt"
	"log"
	"os"
)

// Create a dot graphviz file with db data read form the provided channel.
func DeployMake(c <-chan CmdData, makefilename string) {
	os.Remove(makefilename)

	fd, err := os.Create(makefilename)
	if err != nil {
		log.Fatal(err)
	}

	// Header:
	fd.WriteString(fmt.Sprintf("# Automatically generated from %s\n\n", makefilename))
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
}

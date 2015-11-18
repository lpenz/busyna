package libbusyna

import (
	"fmt"
	"log"
	"os"
)

// Create a dot graphviz file with db data read form the provided channel.
func DeployDot(c <-chan CmdData, dotfilename string) {
	os.Remove(dotfilename)

	fd, err := os.Create(dotfilename)
	if err != nil {
		log.Fatal(err)
	}

	fd.WriteString("digraph {\n\trankdir=LR\n\n")
	i := 0
	found := map[string]bool{}
	for cmddata := range c {
		fd.WriteString(fmt.Sprintf("\t\"node%d\" [ label=\"%s\" ]\n", i, cmddata.Cmd.Line))

		for dep := range cmddata.Deps {
			if _, ok := found[dep]; !ok {
				fd.WriteString(fmt.Sprintf("\t\"%s\" [ shape=rectangle ]\n", dep))
				found[dep] = true
			}
			fd.WriteString(fmt.Sprintf("\t\"%s\" -> node%d\n", dep, i))
		}
		for target := range cmddata.Targets {
			if _, ok := found[target]; !ok {
				fd.WriteString(fmt.Sprintf("\t\"%s\" [ shape=rectangle ]\n", target))
				found[target] = true
			}
			fd.WriteString(fmt.Sprintf("\tnode%d -> \"%s\"\n", i, target))
		}
		i++
	}
	fd.WriteString("}\n")
	fd.Close()
}

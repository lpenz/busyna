package libbusyna

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func DeployRc(c <-chan Cmd, outputfile string) {
	fd, err := ioutil.TempFile(filepath.Dir(outputfile), "busyna.rc-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(fd.Name())

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	rcdir := cwd

	for cmd := range c {
		if cmd.Dir != "." {
			reldir, err := filepath.Rel(rcdir, cmd.Dir)
			if err != nil {
				log.Fatal(err)
			}
			if reldir != "." {
				fd.WriteString(fmt.Sprintf("cd %s\n", reldir))
				rcdir = reldir
			}
		}
		rcdir = cwd
		fd.WriteString(cmd.Line)
		fd.WriteString("\n")
	}

	fd.Close()

	if err = os.Rename(fd.Name(), outputfile); err != nil {
		log.Fatal(err)
	}
}

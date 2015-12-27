package libbusyna

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func DeployRc(c <-chan Cmd, outputfile string) {
	fd, err := ioutil.TempFile(filepath.Dir(outputfile), "busyna.rc-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(fd.Name())
	fd.WriteString("#!/bin/sh\n\nset -e -x\n\n")

	cwd0, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	cwd := cwd0

	env := map[string]string{}

	for cmd := range c {
		// Deal with environment
		for k, v := range cmd.Env {
			v0, found := env[k]
			if !found || v0 != v {
				fd.WriteString(fmt.Sprintf("%s=%v\n", k, v))
				env[k] = v
			}
		}
		for k := range env {
			_, found := cmd.Env[k]
			if !found {
				fd.WriteString(fmt.Sprintf("unset %s\n", k))
				delete(env, k)
			}
		}

		// Deal with directories
		absdir := ""
		if filepath.IsAbs(cmd.Dir) {
			absdir = cmd.Dir
		} else {
			absdir = filepath.Join(cwd0, cmd.Dir)
		}
		// only chdir to a different dir
		if absdir != cwd {
			// let's see if we are currently inside project directory
			var destdir string
			var err error
			if strings.HasPrefix(cwd, cwd0) {
				destdir, err = filepath.Rel(cwd, absdir)
			} else {
				destdir = absdir
				err = error(nil)
			}
			if err != nil {
				log.Fatal(err)
			}
			// destdir has what goes to chdir
			if strings.HasPrefix(absdir, cwd0) {
				// use destdir
				fd.WriteString(fmt.Sprintf("cd %s\n", destdir))
			} else {
				// abs path, fail
				log.Fatalf("busyna.rc should use only relative directories")
			}
			cwd = absdir
		}

		// Write command-line
		fd.WriteString(cmd.Line)
		fd.WriteString("\n")
	}

	fd.Close()

	if err = os.Rename(fd.Name(), outputfile); err != nil {
		log.Fatal(err)
	}
}

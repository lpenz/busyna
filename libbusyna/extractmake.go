package libbusyna

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ExtractShellCreate creates the shell help script that is executed as the
// shell itself.
func ExtractShellCreate(outputfile string) *os.File {
	shfile, err := ioutil.TempFile("", "busyna-shell-")
	if err != nil {
		log.Fatal(err)
	}
	shfile.Chmod(0777)
	o, err := filepath.Abs(outputfile)
	if err != nil {
		log.Fatal(err)
	}
	shfile.WriteString("#!/bin/sh\n\n")
	shfile.WriteString("shift # get rid of -c\n")
	shfile.WriteString("(\n")
	shfile.WriteString("# get rid of standard environment\n")
	shfile.WriteString("unset MAKEFLAGS\n")
	shfile.WriteString("unset MFLAGS\n")
	shfile.WriteString("unset MAKELEVEL\n")
	for _, e := range os.Environ() {
		ev := strings.Split(e, "=")
		shfile.WriteString(fmt.Sprintf("unset %s\n", ev[0]))
	}
	shfile.WriteString(fmt.Sprintf("/usr/bin/env >> %s\n", o))
	shfile.WriteString(")\n")
	shfile.WriteString(fmt.Sprintf("echo cd \"$PWD\" >> %s\n\n", o))
	shfile.WriteString(fmt.Sprintf("echo \"$@\" >> %s\n\n", o))
	shfile.WriteString("exec /bin/sh -c \"$1\"\n\n")
	shfile.Close()
	return shfile
}

// ExtractMake creates the shell script and the Makefile that are used to
// create a busyna.rc from an existing Makefile
func ExtractMake(outputfile string) (*os.File, *os.File) {
	os.Remove(outputfile)

	fd, err := os.Create(outputfile)
	if err != nil {
		log.Fatal(err)
	}

	fd.WriteString("#!/usr/bin/env busyna-sh\n\n")
	shfile := ExtractShellCreate(outputfile)
	//fmt.Printf("tmp %s\n", shfile.Name())

	mkfile, err := ioutil.TempFile("", "busyna-makefile-")
	if err != nil {
		log.Fatal(err)
	}
	mkfile.WriteString(fmt.Sprintf("SHELL:=%s\n", shfile.Name()))
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	mkfile.WriteString(fmt.Sprintf("include %s/Makefile\n", cwd))
	mkfile.Sync()

	return mkfile, shfile
}

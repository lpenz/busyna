package libbusyna

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
	shfile.WriteString(fmt.Sprintf("echo -n '(' >> %s\n\n", o))
	shfile.WriteString(fmt.Sprintf("echo \"$@\" | sed 's@\\\\$@@' | tr -d '\\n' >> %s\n\n", o))
	shfile.WriteString(fmt.Sprintf("echo ')' >> %s\n\n", o))
	shfile.WriteString("exec /bin/sh -c \"$1\"\n\n")
	shfile.Close() // Must close to avoid "text file busy"
	return shfile
}

// ExtractMakefileCreate creates the helper Makefile used to extract make's
// commands.
func ExtractMakefileCreate(shfile *os.File) *os.File {
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
	mkfile.Close()

	return mkfile
}

// ExtractMake creates the shell script and the Makefile that are used to
// create a busyna.rc from an existing Makefile
func ExtractMake(outputfile string) {
	rcfile, err := ioutil.TempFile(filepath.Dir(outputfile), "busyna.rc-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(rcfile.Name())
	rcfile.Close()

	shfile := ExtractShellCreate(rcfile.Name())
	mkfile := ExtractMakefileCreate(shfile)

	cmd := exec.Command("make", "-B", "-j1", "-f", mkfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(fmt.Sprintf("%s while running make %s with shell %s to fill %s - keeping files", err, mkfile.Name(), shfile.Name(), rcfile.Name()))
	}

	DeployRc(RcParse(rcfile.Name(), ChanFromFile(rcfile.Name())), outputfile)

	os.Remove(shfile.Name())
	os.Remove(mkfile.Name())
	os.Remove(rcfile.Name())
}

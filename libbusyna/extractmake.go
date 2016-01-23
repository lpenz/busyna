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
func ExtractShellCreate(rcfile *os.File) *os.File {
	shfile, err := ioutil.TempFile("", "busyna-shell-")
	if err != nil {
		log.Fatal(err)
	}
	shfile.Chmod(0755)
	for _, l := range []string{
		`#!/bin/sh`,
		``,
		`shift # get rid of -c`,
		fmt.Sprintf(`SUBMAKE=$(echo "$@" | sed 's@%s@make SHELL=%s MAKE=%s@')`, rcfile.Name(), shfile.Name(), rcfile.Name()),
		`if [ "$SUBMAKE" != "$(echo "$@")" ]; then # recursive make call`,
		"\t/bin/sh -c \"$SUBMAKE\"",
		"\texit $?",
		`fi`,
		`(`,
		`# get rid of standard environment`,
		`unset MAKE`,
		`unset MAKEOVERRIDES`,
		`unset MAKEFLAGS`,
		`unset MFLAGS`,
		`unset MAKELEVEL`,
	} {
		shfile.WriteString(l)
		shfile.WriteString("\n")
	}
	for _, e := range os.Environ() {
		ev := strings.Split(e, "=")
		shfile.WriteString(fmt.Sprintf("unset %s\n", ev[0]))
	}
	shfile.WriteString(fmt.Sprintf("/usr/bin/env | /bin/sed 's@\\(^[^=]\\+\\)=\\(.*\\)$@\\1='\"'\"'\\2'\"'\"'@' >> %s\n", rcfile.Name()))
	shfile.WriteString(")\n")
	shfile.WriteString(fmt.Sprintf("echo cd \"$PWD\" >> %s\n\n", rcfile.Name()))
	shfile.WriteString(fmt.Sprintf("echo -n '(' >> %s\n\n", rcfile.Name()))
	shfile.WriteString(fmt.Sprintf("echo \"$@\" | sed 's@\\\\$@@' | tr -d '\\n' >> %s\n\n", rcfile.Name()))
	shfile.WriteString(fmt.Sprintf("echo ')' >> %s\n\n", rcfile.Name()))
	//shfile.WriteString("echo exec /bin/sh -c \"$@\" >&2\n\n")
	shfile.WriteString("exec /bin/sh -c \"$@\"\n\n")
	shfile.Close() // Must close to avoid "text file busy"
	return shfile
}

// ExtractMake creates the shell script and the Makefile that are used to
// create a busyna.rc from an existing Makefile
func ExtractMake(outputfile string) {
	outputfileabs, err := filepath.Abs(outputfile)
	if err != nil {
		log.Fatal(err)
	}
	rcfile, err := ioutil.TempFile(filepath.Dir(outputfileabs), "busyna.rc-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(rcfile.Name())
	rcfile.Close()

	shfile := ExtractShellCreate(rcfile)
	defer os.Remove(shfile.Name())

	cmd := exec.Command("make", "-B", "-j1", fmt.Sprintf("SHELL=%s", shfile.Name()), fmt.Sprintf("MAKE=%s", rcfile.Name()))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(fmt.Sprintf("%s while running make with shell %s to fill %s - keeping files", err, shfile.Name(), rcfile.Name()))
	}

	DeployRcCmd(RcParse(rcfile.Name(), ChanFromFile(rcfile.Name())), outputfile)
}

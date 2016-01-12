package libbusyna

import (
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func writeFile(t *testing.T, filename string, contents []string) {
	fd, err := os.Create(filename)
	if err != nil {
		t.Error(err)
	}
	for _, l := range contents {
		fd.WriteString(l)
		fd.WriteString("\n")
	}
	fd.Close()
}

// TestExtractMake tests the make extract function
func TestExtractMake(t *testing.T) {
	makefile := []string{
		`all:`,
		"\tTEST=test\\ with\\ spaces $(MAKE) -f mkfile",
	}
	mkfile := []string{
		`all: file1.txt file2.txt`,
		``,
		`# create a file`,
		`file1.txt:`,
		"\techo asdf > $@",
		``,
		`# copy it to another two files in the same target`,
		`file2.txt: file1.txt`,
		"\tcat file1.txt > $@",
		"\tcat file1.txt > file3.txt",
		``,
	}
	defer func() {
		for _, f := range []string{
			"mkfile",
			"Makefile",
			"test.rc",
			// These also test if the files were created:
			"file1.txt",
			"file2.txt",
			"file3.txt",
		} {
			if err := os.Remove(f); err != nil {
				t.Error(err)
			}
		}
	}()
	// Write the Makefile:
	writeFile(t, "Makefile", makefile)
	writeFile(t, "mkfile", mkfile)
	// Extract it:
	ExtractMake("test.rc")
	// Remove the created targets
	os.Remove("file1.txt")
	os.Remove("file2.txt")
	os.Remove("file3.txt")
	// Run the created test.rc with the shell.
	// The defer's above check if the files are created.
	// That also checks busyna shell compatibility.
	cmd := exec.Command("/bin/sh", "test.rc")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Errorf("%s while running test.rc with shell", err)
	}
	// Check the answer:
	env0 := map[string]string{
		`TEST`: `test with spaces`,
	}
	ans := []Cmd{
		Cmd{`(echo asdf > file1.txt)`, env0, `.`, nil},
		Cmd{`(cat file1.txt > file2.txt)`, env0, `.`, nil},
		Cmd{`(cat file1.txt > file3.txt)`, env0, `.`, nil},
	}
	i := 0
	for cmd := range RcParse("", ChanFromFile("test.rc")) {
		if !reflect.DeepEqual(cmd, ans[i]) {
			t.Errorf("cmd %d mismatch: %v != %v", i, cmd, ans[i])
		}
		i++
	}
}

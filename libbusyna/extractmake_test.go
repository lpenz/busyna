package libbusyna

import (
	"os"
	"os/exec"
	"testing"
)

// TestExtractMake tests the make deploy function
func TestExtractMake(t *testing.T) {
	makefile := []string{
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
		if err := os.Remove("Makefile"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("test.rc"); err != nil {
			t.Error(err)
		}
		// These also test if the files were created:
		if err := os.Remove("file1.txt"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("file2.txt"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("file3.txt"); err != nil {
			t.Error(err)
		}
	}()
	// Write the Makefile:
	fd, err := os.Create("Makefile")
	if err != nil {
		t.Error(err)
	}
	for _, l := range makefile {
		fd.WriteString(l)
		fd.WriteString("\n")
	}
	fd.Close()
	// Extract it:
	ExtractMake("test.rc")
	// Remove the created targets
	os.Remove("file1.txt")
	os.Remove("file2.txt")
	os.Remove("file3.txt")
	// Run the created test.rc with the shell.
	// The defer's above check if the files are created.
	exec.Command("/bin/sh", "test.rc").Run()
	// That also checks busyna shell compatibility.
}

package libbusyna

import (
	"os"
	"os/exec"
	"testing"
)

// exists checks if a file exists
func exists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// TestDeployMake tests the make deploy function
func TestDeployMake(t *testing.T) {
	busynarc := []string{
		`# create a file`,
		`echo asdf > file1.txt`,
		`# copy it to another two files in the same command`,
		`cat file1.txt > file2.txt; cat file1.txt > file3.txt`,
	}
	defer func() {
		if err := os.Remove("test.db"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("Makefile"); err != nil {
			t.Error(err)
		}
	}()
	// Write the database:
	DbWrite(RcRun(RcParse("", ChanFromList(busynarc))), "test.db")
	// Remove target files:
	if err := os.Remove("file1.txt"); err != nil {
		t.Error(err)
	}
	if err := os.Remove("file2.txt"); err != nil {
		t.Error(err)
	}
	// Create Makefile
	DeployMake(DbRead("test.db"), "Makefile")
	// Test make
	if err := exec.Command("make").Run(); err != nil {
		t.Error(err)
	}
	// Let's see if the files are there:
	if !exists("file1.txt") {
		t.Error("make target not found")
	}
	if !exists("file2.txt") {
		t.Error("make target not found")
	}
	// Test make clean
	if err := exec.Command("make", "clean").Run(); err != nil {
		t.Error(err)
	}
	if exists("file1.txt") {
		t.Error("make clean did not rm target")
	}
	if exists("file2.txt") {
		t.Error("make clean did not rm target")
	}
}

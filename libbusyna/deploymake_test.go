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
		`# create a dir, cd and write a file`,
		`mkdir -p d`,
		`cd d`,
		`echo > t`,
	}
	defer func() {
		for _, f := range []string{`test.db`, `Makefile`} {
			if err := os.Remove(f); err != nil {
				t.Error(err)
			}
		}
	}()
	// Write the database:
	DbWrite(RcRun(RcParse("", ChanFromList(busynarc))), "test.db")
	// Remove target files:
	for _, f := range []string{`file1.txt`, `file2.txt`, `d/t`, `d`} {
		if err := os.Remove(f); err != nil {
			t.Error(err)
		}
	}
	// Create Makefile
	DeployMake(DbRead("test.db"), "Makefile")
	// Test make
	if err := exec.Command("make").Run(); err != nil {
		t.Error(err)
	}
	// Let's see if the files are there:
	for _, f := range []string{`file1.txt`, `file2.txt`, `d/t`, `d`} {
		if !exists(f) {
			t.Errorf("make target %s not found", f)
		}
	}
	// Test make clean
	if err := exec.Command("make", "clean").Run(); err != nil {
		t.Error(err)
	}
	for _, f := range []string{`file1.txt`, `file2.txt`, `d/t`, `d`} {
		if exists(f) {
			t.Errorf("make clean did not rm target %s", f)
		}
	}
}

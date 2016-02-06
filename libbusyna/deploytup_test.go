package libbusyna

import (
	"os"
	"os/exec"
	"testing"
)

// TestDeployTup tests the make deploy function
func TestDeployTup(t *testing.T) {
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
		for _, f := range []string{`test.db`, `Tupfile.lua`, `file1.txt`, `file2.txt`, `file3.txt`, `d/t`, `d`} {
			if err := os.Remove(f); err != nil {
				t.Error(err)
			}
		}
		for _, f := range []string{`.tup`} {
			if err := os.RemoveAll(f); err != nil {
				t.Error(err)
			}
		}
	}()
	// Write the database:
	DbWrite(RcRun(RcParse("", ChanFromList(busynarc))), "test.db")
	// Remove target files:
	for _, f := range []string{`file1.txt`, `file2.txt`, `file3.txt`, `d/t`, `d`} {
		if err := os.Remove(f); err != nil {
			t.Error(err)
		}
	}
	// Create Tupfile.lua
	DeployTup(DbRead("test.db"), "Tupfile.lua")
	// Test tup
	r := exec.Command("tup", "init")
	r.Stdout = os.Stdout
	r.Stderr = os.Stderr
	if err := r.Run(); err != nil {
		t.Error(err)
	}
	r = exec.Command("tup")
	r.Stdout = os.Stdout
	r.Stderr = os.Stderr
	if err := r.Run(); err != nil {
		t.Error(err)
	}
	// Let's see if the files are there:
	for _, f := range []string{`file1.txt`, `file2.txt`, `file3.txt`, `d/t`, `d`} {
		if !exists(f) {
			t.Errorf("make target %s not found", f)
		}
	}
}

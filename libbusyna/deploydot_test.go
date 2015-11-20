package libbusyna

import (
	"os"
	"os/exec"
	"testing"
)

// TestDeployDot tests the graphviz deploy function
func TestDeployDot(t *testing.T) {
	busynarc := []string{
		`# create a file`,
		`echo asdf > file1.txt`,
		`# copy it to another`,
		`cat file1.txt > file2.txt`,
	}
	defer func() {
		if err := os.Remove("file1.txt"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("file2.txt"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("test.db"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("test.dot"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("test.png"); err != nil {
			t.Error(err)
		}
	}()
	// Write the database:
	DbWrite(RcRun(RcParse("", ChanFromList(busynarc))), "test.db")
	// Let's see if the dot function generates a valid file:
	DeployDot(DbRead("test.db"), "test.dot")
	if err := exec.Command("dot", "-Tpng", "test.dot", "-o", "test.png").Run(); err != nil {
		t.Error(err)
	}
}

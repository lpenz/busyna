package libbusyna

import (
	"io/ioutil"
	"os"
	"testing"
)

// TestDeployRc tests the busyna.rc deploy function
func TestDeployRc(t *testing.T) {
	busynarc := []string{
		`# create a file`,
		`echo asdf > file1.txt`,
		`# copy it to another two files in the same command`,
		`cat file1.txt > file2.txt; cat file1.txt > file3.txt`,
	}

	// Create answer from the file above:
	ans := []Cmd{}
	for c := range RcParse("", ChanFromList(busynarc)) {
		ans = append(ans, c)
	}

	// Create tmp rc file and compare the parser results:
	fd, err := ioutil.TempFile("", "busyna.rc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fd.Name())
	DeployRc(RcParse("", ChanFromList(busynarc)), fd.Name())
	cmds := []Cmd{}
	for c := range RcParse("", ChanFromFile(fd.Name())) {
		cmds = append(cmds, c)
	}

	// Compare:
	if len(ans) != len(cmds) {
		t.Errorf("len mismatch: len(dat)=%d != len(ans)=%d", len(cmds), len(ans))
	}
	for i := 0; i < len(ans); i++ {
		if !CmdEqual(cmds[i], ans[i]) {
			t.Errorf("i %d Cmd mismatch: %v != %v", i, cmds[i], ans[i])
		}
	}
}

package libbusyna

import (
	"io/ioutil"
	"os"
	"testing"
)

// TestDeployRc tests the busyna.rc deploy function
func TestDeployRcCmd(t *testing.T) {
	busynarc := []string{
		`ls 0`,
		// Environment, and then command:
		`e=5`,
		`ls 1`,
		// Remove environment, repeat command:
		`unset e`,
		`ls 2`,
		// Go to a subdir, run two commands:
		`cd sub1`,
		`ls 3`,
		`ls 4`,
		// Go to a subsubdir, run a commands:
		`cd sub11`,
		`ls 5`,
		// Go back to a dir that is a child from top:
		`cd ../../sub2`,
		`ls 6`,
		// Go back to the previous directory
		`cd -`,
		`ls 7`,
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
	DeployRcCmd(RcParse("", ChanFromList(busynarc)), fd.Name())
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

// TestDeployRc checks that DeployRc skips lines that touch no files
func TestDeployRc(t *testing.T) {
	busynarc := []string{
		`echo asdf`,
		`echo > asdf`,
	}
	defer func() {
		for _, f := range []string{`test.db`, `busyna.rc`} {
			if err := os.Remove(f); err != nil {
				t.Error(err)
			}
		}
	}()
	// Write the database:
	DbWrite(RcRun(RcParse("", ChanFromList(busynarc))), "test.db")
	// Remove target files:
	for _, f := range []string{`asdf`} {
		if err := os.Remove(f); err != nil {
			t.Error(err)
		}
	}
	// Create busyna.rc
	DeployRc(DbRead("test.db"), "busyna.rc")
	// Parse busyna.rc, check number of commands
	ans := []Cmd{}
	for c := range RcParse("", ChanFromFile(`busyna.rc`)) {
		ans = append(ans, c)
	}
	if len(ans) != 1 {
		t.Fatal(`expecting single command`)
	}
}

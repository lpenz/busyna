package libbusyna

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

var env0 = map[string]string{}
var fileset0 = map[string]bool{}

// rctolist parses the provided busyna.rc string list into a Cmd list
func rctolist(busynarc []string) []Cmd {
	cmds := []Cmd{}
	for cmddata := range RcParse("", ChanFromList(busynarc)) {
		cmds = append(cmds, cmddata)
	}
	return cmds
}

// TestRcParser tests some basic parser properties
func TestRcParser(t *testing.T) {
	busynarc := []string{
		// Comments:
		`# asdf`,
		`#zxcv`,
		// Empty line skipping:
		``,
		` `,
		`set -x -e`,
		`set -e  -x`,
		`set -e  -x	-o pipefail`,
		"\t",
		// Simple command, no env:
		`echo zxcv _2 999`,
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
		// Go back to top:
		`cd ..`,
		// Go to an absolute path:
		`cd /`,
		`ls 7`,
		`ls 8`,
		// Go back to the previous directory
		`cd -`,
		`ls 9`,
		// Some weird commands:
		`cmd =5`,
		`e='5 9'`,
		`ls 10`,
	}
	ans := []Cmd{
		Cmd{`echo zxcv _2 999`, env0, `.`, nil},
		Cmd{`ls 1`, map[string]string{`e`: `5`}, `.`, nil},
		Cmd{`ls 2`, env0, `.`, nil},
		Cmd{`ls 3`, env0, `sub1`, nil},
		Cmd{`ls 4`, env0, `sub1`, nil},
		Cmd{`ls 5`, env0, `sub1/sub11`, nil},
		Cmd{`ls 6`, env0, `sub2`, nil},
		Cmd{`ls 7`, env0, `/`, errors.New("busyna.rc should use only relative directories")},
		Cmd{`ls 8`, env0, `/`, errors.New("busyna.rc should use only relative directories")},
		Cmd{`ls 9`, env0, `.`, nil},
		Cmd{`cmd =5`, env0, `.`, nil},
		Cmd{`ls 10`, map[string]string{`e`: `5 9`}, `.`, nil},
	}
	cmds := rctolist(busynarc)
	if len(ans) != len(cmds) {
		t.Errorf("len mismatch: len(%s)=%d != len(%s)=%d", cmds, len(cmds), ans, len(ans))
	}
	for i := 0; i < len(ans); i++ {
		if !reflect.DeepEqual(cmds[i], ans[i]) {
			t.Errorf("cmd %d mismatch: %v != %v", i, cmds[i], ans[i])
		}
	}
}

// TestRcRun tests the run function with a dummy backend.
func TestRcRun(t *testing.T) {
	busynarc := []string{
		`# create a file`,
		`echo asdf > file1.txt`,
		`# copy it to another`,
		`cat file1.txt > file2.txt`,
	}
	ans := []CmdData{
		CmdData{
			Cmd{`echo asdf > file1.txt`, env0, `.`, nil},
			fileset0,
			map[string]bool{`file1.txt`: true},
		},
		CmdData{
			Cmd{`cat file1.txt > file2.txt`, env0, `.`, nil},
			map[string]bool{`file1.txt`: true},
			map[string]bool{`file2.txt`: true},
		},
	}
	cmddatas := []CmdData{}
	defer func() {
		if err := os.Remove("file1.txt"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("file2.txt"); err != nil {
			t.Error(err)
		}
	}()
	for cmddata := range RcRun(RcParse("", ChanFromList(busynarc))) {
		cmddatas = append(cmddatas, cmddata)
	}
	if len(ans) != len(cmddatas) {
		t.Errorf("len mismatch: len(dat)=%d != len(ans)=%d", len(cmddatas), len(ans))
	}
	for i := 0; i < len(ans); i++ {
		if !CmdDataEqual(cmddatas[i], ans[i], false) {
			t.Errorf("i %d Cmd mismatch: %v != %v", i, cmddatas[i], ans[i])
		}
	}
}

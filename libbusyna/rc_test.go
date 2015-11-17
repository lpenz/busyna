// Test rc functions
package libbusyna

import (
	"os"
	"reflect"
	"testing"
)

var env0 = map[string]string{}
var fileset0 = map[string]bool{}

// TestBusynarcParser tests some basic parser properties
func TestBusynarcParser(t *testing.T) {
	busynarc := []string{
		`# asdf`,
		`#zxcv`,
		`cmd 1`,
		``,
		`cmd zxcv _2`,
		` `,
		`tst=5`,
		`cmd =5`,
		`tst=8`,
		`cmd`,
		`tst=`,
		`_asdf`,
	}
	ans := []Cmd{
		Cmd{`cmd 1`, env0, ``},
		Cmd{`cmd zxcv _2`, env0, ``},
		Cmd{`cmd =5`, map[string]string{`tst`: `5`}, ``},
		Cmd{`cmd`, map[string]string{`tst`: `8`}, ``},
		Cmd{`_asdf`, map[string]string{`tst`: ``}, ``},
	}
	cmds := []Cmd{}
	for c := range Parse(ChanFromList(busynarc)) {
		cmds = append(cmds, c)
	}
	if len(ans) != len(cmds) {
		t.Errorf("len mismatch: len(%s)=%d != len(%s)=%d", cmds, len(cmds), ans, len(ans))
	}
	for i := 0; i < len(ans); i++ {
		if !reflect.DeepEqual(cmds[i], ans[i]) {
			t.Errorf("arg %d mismatch: %s != %s", i, cmds[i], ans[i])
		}
	}
}

// TestBusynarcRun tests the run function with a dummy backend.
func TestBusynarcRun(t *testing.T) {
	busynarc := []string{
		`# create a file`,
		`echo asdf > file1.txt`,
		`# copy it to another`,
		`cat file1.txt > file2.txt`,
	}
	ans := []CmdData{
		CmdData{
			Cmd{`echo asdf > file1.txt`, env0, ``},
			fileset0,
			map[string]bool{`file1.txt`: true},
		},
		CmdData{
			Cmd{`cat file1.txt > file2.txt`, env0, ``},
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
	for cmddata := range Run(Parse(ChanFromList(busynarc))) {
		cmddatas = append(cmddatas, cmddata)
	}
	if len(ans) != len(cmddatas) {
		t.Errorf("len mismatch: len(dat)=%d != len(ans)=%d", len(cmddatas), len(ans))
	}
	for i := 0; i < len(ans); i++ {
		if !reflect.DeepEqual(cmddatas[i].Cmd, ans[i].Cmd) {
			t.Errorf("i %d Cmd mismatch: %s != %s", i, cmddatas[i].Cmd, ans[i].Cmd)
		}
		for j := range ans[i].Deps {
			if _, ok := cmddatas[i].Deps[j]; !ok {
				t.Errorf("i %d dep %s not found", i, j)
			}
		}
		for j := range ans[i].Targets {
			if _, ok := cmddatas[i].Targets[j]; !ok {
				t.Errorf("i %d target %s not found", i, j)
			}
		}
	}
}

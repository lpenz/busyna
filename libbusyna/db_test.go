package libbusyna

import (
	"os"
	"testing"
)

// TestBusynarcRun tests the run function with a dummy backend.
func TestDb(t *testing.T) {
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
	}()
	// Get the answer by running the busynarc above
	ans := []CmdData{}
	for cmddata := range RcRun(RcParse(ChanFromList(busynarc))) {
		ans = append(ans, cmddata)
	}
	// Write it to the database:
	DbWrite(RcRun(RcParse(ChanFromList(busynarc))), "test.db")
	cmddatas := []CmdData{}
	for cmddata := range DbRead("test.db") {
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

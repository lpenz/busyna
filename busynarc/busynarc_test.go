package busynarc

import (
	"reflect"
	"testing"

	"github.com/lpenz/busyna/misc"
)

// TestBusynarc1 tests some basic parser properties
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
	ans := []Command{
		Command{map[string]string{}, ``, `cmd 1`},
		Command{map[string]string{}, ``, `cmd zxcv _2`},
		Command{map[string]string{`tst`: `5`}, ``, `cmd =5`},
		Command{map[string]string{`tst`: `8`}, ``, `cmd`},
		Command{map[string]string{`tst`: ``}, ``, `_asdf`},
	}
	cmds := []Command{}
	for c := range Parse(misc.ChanFromList(busynarc)) {
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

package libbusyna

import (
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// TestStraceRun tests basic strace execution.
func TestStraceRun(t *testing.T) {
	StraceRun("echo asdf > /dev/null", nil, "")
}

// Example strace file used to test parsers.
var straceout = []string{
	`16819 stat64("/usr/bin/unbuffer", {st_mode=S_IFREG|0755, st_size=640, ...}) = 0`,
	`16819 stat64("/usr/bin/unbuffer", {st_mode=S_IFREG|0755, st_size=640, ...}) = 0`,
	`16819 clone(child_stack=0, flags=CLONE_CHILD_CLEARTID|CLONE_CHILD_SETTID|SIGCHLD, child_tidptr=0xf74ee728) = 16820`,
	`16819 stat64("/home/lpenz/bin/colorize", {st_mode=S_IFREG|0755, st_size=1483, ...}) = 0`,
	`16819 clone( <unfinished ...>`,
	`16820 execve("/usr/bin/unbuffer", ["unbuffer", "scons", "--max-drift=1", "--implicit-cache", "--debug=explain"], [/* 34 vars */] <unfinished ...>`,
	`16819 <... clone resumed> child_stack=0, flags=CLONE_CHILD_CLEARTID|CLONE_CHILD_SETTID|SIGCHLD, child_tidptr=0xf74ee728) = 16821`,
	`16820 <... execve resumed> )            = 0`,
	`16820 open("/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3`,
	`16821 execve("/home/lpenz/bin/colorize", ["colorize", "^\\(\\s\\+new:\\s\\|\\s\\+old:\\s\\|scons"...], [/* 34 vars */] <unfinished ...>`,
	`16820 open("/lib/i386-linux-gnu/i686/cmov/libc.so.6", O_RDONLY|O_CLOEXEC) = 3`,
	`16821 <... execve resumed> )            = 0`,
	`16821 open("/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3`,
	`16821 open("/lib/i386-linux-gnu/libtinfo.so.5", O_RDONLY|O_CLOEXEC) = 3`,
	`16821 open("/lib/i386-linux-gnu/i686/cmov/libdl.so.2", O_RDONLY|O_CLOEXEC) = 3`,
	`16820 stat64("/home/lpenz/projs/lpenz.github.com", {st_mode=S_IFDIR|0755, st_size=4096, ...}) = 0`,
	`16820 stat64(".", {st_mode=S_IFDIR|0755, st_size=4096, ...}) = 0`,
	`16820 open("/usr/bin/unbuffer", O_RDONLY <unfinished ...>`,
	`16821 open("/lib/i386-linux-gnu/i686/cmov/libc.so.6", O_RDONLY|O_CLOEXEC <unfinished ...>`,
	`16820 <... open resumed> )              = 3`,
	`16821 <... open resumed> )              = 3`,
	`16820 execve("/home/lpenz/bin/tclsh", ["tclsh", "/usr/bin/unbuffer", "scons", "--max-drift=1", "--implicit-cache", "--debug=explain"], [/* 34 vars */]) = -1 ENOENT (No such file or directory)`,
	`16820 ????( <unavailable>)= ? <unavailable>`,
	`16820 execve("/usr/local/bin/tclsh", ["tclsh", "/usr/bin/unbuffer", "scons", "--max-drift=1", "--implicit-cache", "--debug=explain"], [/* 34 vars */]) = -1 ENOENT (No such file or directory)`,
	`16820 --- SIGCHLD (Child exited) @ 0 (0) ---`,
	`16820 execve("/usr/bin/tclsh", ["tclsh", "/usr/bin/unbuffer", "scons", "--max-drift=1", "--implicit-cache", "--debug=explain"], [/* 34 vars */]) = 0`,
	`16820 execve("/usr/bin/tclsh", ["tclsh", "/usr/bin/unbuffer", "scons", "--max-drift=1", "--implicit-cache", "--debug=explain"], [/* 34 vars */]) = ? <unavailable>`,
	`16820 ????(= ? <unavailable>`,
	`16821 open("/dev/tty", O_RDWR|O_NONBLOCK|O_LARGEFILE) = 3`,
	`16820 open("/etc/ld.so.cache", O_RDONLY|O_CLOEXEC <unfinished ...>`,
	`16820 <... ???? resumed> ) = ? <unavailable>`,
	`16821 open("/usr/lib/locale/locale-archive", O_RDONLY|O_LARGEFILE|O_CLOEXEC <unfinished ...>`,
	`16820 <... open resumed> )              = 3`,
	`16821 <... open resumed> )              = 3`,
	`16820 open("/usr/lib\"/libtcl8.5.so.0", O_RDONLY|O_CLOEXEC) = 3`,
	`16820 exit_group(0)                     = ?`,
}

// TestStraceParse1 tests strace level1 parser (joining) by counting and
// checking strings.
func TestStraceParse1(t *testing.T) {
	// Count strings that will be parsed away by StraceParser1
	n := len(straceout)
	for _, l := range straceout {
		if strings.Contains(l, "resumed") || strings.Contains(l, "--- SIG") {
			n--
			continue
		}
		m, err := regexp.MatchString("^[0-9]+ \\?.*", l)
		if err != nil {
			t.Error(err)
		}
		if m {
			n--
			continue
		}
	}
	if n == len(straceout) {
		t.Error("test string has no level 1 parser tokens")
	}

	// Parse, and check that they went away and that the count is right
	parsed := make([]string, 0, len(straceout))
	for l := range StraceParse1(ChanFromList(straceout)) {
		if strings.Contains(l, "resumed") || strings.Contains(l, "finished") {
			t.Error("found invalid string in parsed results: " + l)
		}
		parsed = append(parsed, l)
	}
	if len(parsed) != n {
		t.Error("incorrect len of parsed strings")
	}
}

// TestStraceParse2Basic tests strace level2 parser by counting parsed entities.
func TestStraceParse2Basic(t *testing.T) {
	nopen := 0
	nexec := 0
	for _, l := range straceout {
		if strings.Contains(l, " open(") {
			nopen++
		}
		if strings.Contains(l, " execve(") {
			nexec++
		}
	}
	syscalls := map[string]int{}
	for info := range StraceParse2(StraceParse1(ChanFromList(straceout))) {
		syscalls[info.syscall]++
	}
	if nopen != syscalls["open"] {
		t.Errorf("\"open\" count mismatch: %d != %d", nopen, syscalls["open"])
	}
	if nexec != syscalls["execve"] {
		t.Errorf("\"execve\" count mismatch: %d != %d", nexec, syscalls["execve"])
	}
}

// TestStraceParse2Args tests strace level2 argument splitting.
func TestStraceParse2Args(t *testing.T) {
	tests := []struct {
		str string
		ans []string
	}{
		{"asdf", []string{"asdf"}},
		{"as, df", []string{"as", "df"}},
		{"a {s, d} f", []string{"a {s, d} f"}},
		{"{as, df}", []string{"{as, df}"}},
		{`"as, df"`, []string{`"as, df"`}},
		{`"as, df", gh`, []string{`"as, df"`, "gh"}},
		{`"as, df\", gh"`, []string{`"as, df\", gh"`}},
		{`"as, df\""`, []string{`"as, df\""`}},
	}
	for _, tst := range tests {
		a := StraceParse2Argsplit(tst.str)
		if !reflect.DeepEqual(a, tst.ans) {
			t.Error(a, "!=", tst.ans)
		}
	}
}

// TestStraceParse2Lines tests a specific line-parsing.
func TestStraceParse2Lines(t *testing.T) {
	c := make(chan string)
	go func() {
		defer close(c)
		c <- `16821 open("/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3`
	}()
	for info := range StraceParse2(c) {
		tests := []struct {
			ok  bool
			str string
		}{
			{info.pid == 16821, "pid mismatch"},
			{info.syscall == "open", "syscall mismatch"},
			{info.result == 3, "result mismatch"},
			{info.body == `"/etc/ld.so.cache", O_RDONLY|O_CLOEXEC`, "body mismatch"},
			{info.err == "", "error mismatch"},
		}
		for _, tst := range tests {
			if !tst.ok {
				t.Error(tst.str)
			}
		}
		ans := []string{
			`"/etc/ld.so.cache"`,
			`O_RDONLY|O_CLOEXEC`,
		}
		if len(ans) != len(info.args) {
			t.Errorf("args len mismatch: len(%s)=%d != len(%s)=%d", info.args, len(info.args), ans, len(ans))
		}
		for i := 0; i < len(info.args); i++ {
			if ans[i] != info.args[i] {
				t.Errorf("arg %d mismatch", i)
			}
		}
	}
}

// TestStraceParse3 tests StraceParse3 and StraceParse2
func TestStraceParse3(t *testing.T) {
	c := make(chan string)
	go func() {
		defer close(c)
		c <- `16821 open("/etc/ld.so.cache", O_RDONLY|O_CLOEXEC) = 3`
		c <- `16821 open("w", O_WRONLY|O_CREAT|O_TRUNC|O_CLOEXEC) = 4`
		c <- `16821 open("r", O_RDONLY|O_CLOEXEC) = 5`
		c <- `16821 open("rw", O_RDWR|O_NONBLOCK) = 6`
		c <- `16821 creat("c", 01)                          = 6`
	}()
	r, w := StraceParse3(StraceParse2(c), "")
	rok := map[string]bool{
		"/etc/ld.so.cache": true,
		"r":                true,
		"rw":               true,
	}
	if !reflect.DeepEqual(r, rok) {
		t.Error(r, "!=", rok)
	}
	wok := map[string]bool{
		"w":  true,
		"c":  true,
		"rw": true,
	}
	if !reflect.DeepEqual(w, wok) {
		t.Error(w, "!=", wok)
	}
}

// Test real applications:

// straceRbase has the base read files for the OS where the tests are run.
var straceRbase map[string]bool

// empty is an empty map
var empty = map[string]bool{}

// filetraceTest is the primitive test function that runs the provided command
// and checks if the set of files read and written match the ones provided.
func filetraceTest(t *testing.T, cmd string, dir string, rok map[string]bool, wok map[string]bool) {
	rt, wt := FileTrace(cmd, nil, dir)
	if len(straceRbase) == 0 {
		straceRbase, _ = FileTrace("", nil, "")
	}
	rtst := map[string]bool{}
	for r := range rok {
		rtst[r] = true
	}
	for r := range straceRbase {
		rtst[r] = true
	}
	if !reflect.DeepEqual(rtst, rtst) {
		t.Error("r", rt, "!=", rtst)
	}
	if !reflect.DeepEqual(wt, wok) {
		t.Error("w", wt, "!=", wok)
	}
}

// TestFiletraceEchocat is the base test of read/write that runs an echo with the
// output redirected to a file, and a cat that reads that file.
func TestFiletraceEchocat(t *testing.T) {
	empty := map[string]bool{}
	filetraceTest(t,
		"echo asdf > t",
		"",
		empty,
		map[string]bool{"t": true})
	defer func() {
		if err := os.Remove("t"); err != nil {
			t.Error(err)
		}
	}()
	filetraceTest(t,
		"cat t > h",
		"",
		map[string]bool{"t": true},
		map[string]bool{"h": true})
	defer func() {
		if err := os.Remove("h"); err != nil {
			t.Error(err)
		}
	}()
	filetraceTest(t,
		"cp t j",
		"",
		map[string]bool{"t": true},
		map[string]bool{"j": true})
	defer func() {
		if err := os.Remove("j"); err != nil {
			t.Error(err)
		}
	}()
}

// TestFiletraceChdir tests directory chaging.
func TestFiletraceChdir(t *testing.T) {
	filetraceTest(t,
		"mkdir d; cd d; echo asdf > t",
		"",
		empty,
		map[string]bool{"d/t": true})
	defer func() {
		if err := os.Remove("d/t"); err != nil {
			t.Error(err)
		}
		if err := os.Remove("d"); err != nil {
			t.Error(err)
		}
	}()
}

// TestFiletraceEnv tests the environment argument.
func TestFiletraceEnv(t *testing.T) {
	FileTrace("env > e.txt", map[string]string{"x": "y"}, "")
	defer func() {
		if err := os.Remove("e.txt"); err != nil {
			t.Error(err)
		}
	}()

	data, err := ioutil.ReadFile("e.txt")
	if err != nil {
		t.Fatal(err)
	}
	datastr := string(data)
	if !strings.Contains(datastr, "x=y") {
		t.Fatalf("environment x=y not found in %s", datastr)
	}
}

// TestFiletraceDir tests the dir argument.
func TestFiletraceDir(t *testing.T) {
	os.Mkdir("d", 0755)
	filetraceTest(t,
		"mkdir -p s/ss; cd s; cd ss; echo asdf > t; echo zxcv > z; rm z",
		"d",
		empty,
		map[string]bool{"s/ss/t": true})
	defer func() {
		for _, f := range []string{"d/s/ss/t", "d/s/ss", "d/s", "d"} {
			if err := os.Remove(f); err != nil {
				t.Error(err)
			}
		}
	}()
}

// TestFiletraceRename tests renaming
func TestFiletraceRename(t *testing.T) {
	empty := map[string]bool{}
	filetraceTest(t,
		"echo asdf > t; mv t v",
		"",
		empty,
		map[string]bool{"v": true})
	defer func() {
		if err := os.Remove("v"); err != nil {
			t.Error(err)
		}
	}()
}

package filetrace

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// TestStraceRun tests basic strace execution
func TestStraceRun(t *testing.T) {
	StraceRun("echo asdf > /dev/null", nil, "")
}

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

// straceout_iterate writes straceout to a channel, one line at a time.
func straceout_iterate() <-chan string {
	c := make(chan string)
	go func() {
		defer close(c)
		for _, l := range straceout {
			c <- l
		}
	}()
	return c
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
			t.Fatal(err)
		}
		if m {
			n--
			continue
		}
	}
	if n == len(straceout) {
		t.Fatal("test string has no level 1 parser tokens")
	}

	// Parse, and check that they went away and that the count is right
	parsed := make([]string, 0, len(straceout))
	for l := range StraceParse1(straceout_iterate()) {
		if strings.Contains(l, "resumed") || strings.Contains(l, "finished") {
			t.Fatal("found invalid string in parsed results: " + l)
		}
		parsed = append(parsed, l)
	}
	if len(parsed) != n {
		t.Fatal("incorrect len of parsed strings")
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
	for info := range StraceParse2(StraceParse1(straceout_iterate())) {
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
		a := StraceParse2_argsplit(tst.str)
		if !reflect.DeepEqual(a, tst.ans) {
			t.Fatal(a, "!=", tst.ans)
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
		c <- `16821 creat("c", 01)                          = 6`

	}()
	r, w := StraceParse3(StraceParse2(c))
	rok := map[string]bool{
		"/etc/ld.so.cache": true,
		"r":                true,
	}
	if !reflect.DeepEqual(r, rok) {
		t.Fatal(r, "!=", rok)
	}
	wok := map[string]bool{
		"w": true,
		"c": true,
	}
	if !reflect.DeepEqual(w, wok) {
		t.Fatal(w, "!=", wok)
	}
}

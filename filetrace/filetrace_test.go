package filetrace

import (
	"regexp"
	"strings"
	"testing"
)

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

func TestStraceParse1(t *testing.T) {
	c := make(chan string)
	go func() {
		for _, l := range straceout {
			c <- l
		}
		close(c)
	}()

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
	for l := range StraceParse1(c) {
		if strings.Contains(l, "resumed") || strings.Contains(l, "finished") {
			t.Fatal("found invalid string in parsed results: " + l)
		}
		parsed = append(parsed, l)
	}
	if len(parsed) != n {
		t.Fatal("incorrect len of parsed strings")
	}
}

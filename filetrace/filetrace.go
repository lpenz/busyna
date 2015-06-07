// Package filetrace runs a command and traces the files read and written.
package filetrace

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Misc: #####################################################################

// tmpend finishes a temporary file by closing its file descriptor and deleting
// it in the filesystem.
func tmpend(f *os.File) {
	if f == nil {
		return
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	if err := os.Remove(f.Name()); err != nil {
		log.Fatal(err)
	}
	f = nil
}

func re_findmap(re *regexp.Regexp, l string) map[string]string {
	m := re.FindAllStringSubmatch(l, -1)[0]
	names := re.SubexpNames()
	r := make(map[string]string)
	for i := 0; i < len(names); i++ {
		r[names[i]] = m[i]
	}
	return r
}

// strace runner: ############################################################

// StraceRun runs a command using strace and writes the trace information to
// the returned channel.
// The environment variables and working directory can be specified.
func StraceRun(command string, env []string, dir string) <-chan string {
	cmdfile, err := ioutil.TempFile("", "busyna-strace-cmdfile-")
	cmdfile.WriteString(command)
	cmdfile.WriteString(fmt.Sprint("\n\n\nrm -f ", cmdfile.Name(), "\n"))
	straceout, err := ioutil.TempFile("", "busyna-strace-output-")
	if err != nil {
		log.Fatal(err)
	}
	defer tmpend(straceout)

	a := []string{
		"strace",
		"-f",
		"-a0",
		"-s1024",
		"-etrace=execve,clone,vfork,chdir,creat,unlink,unlinkat,rename,mkdir,rmdir,open",
		"-o" + straceout.Name(),
		"/bin/sh",
		cmdfile.Name(),
	}

	p, err := exec.LookPath("strace")
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Cmd{
		Path:   p,
		Args:   a,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Dir:    dir,
		Env:    env,
	}

	if err = cmd.Run(); err != nil {
		log.Fatal(err)
	}

	// Re-open the file to pass the second fd to the closure.
	// The first is closed by defer.
	fd, err := os.Open(straceout.Name())
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan string)
	go func() {
		defer fd.Close()
		defer close(c)
		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			c <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}()
	return c
}

// strace level 1 parser: ####################################################

var straceparse1_basre = regexp.MustCompile(`^(?P<pid>\d+)\s+`)
var straceparse1_sigre = regexp.MustCompile(`^(?P<pid>\d+)\s+--- (?P<signal>[A-Z]+) \(([a-zA-Z ]+)\) @ [0-9]+ \([0-9]+\) ---$`)
var straceparse1_inire = regexp.MustCompile(`^(?P<pid>\d+)(?P<ini>\s+[^(]+\(.*) <unfinished ...>$`)
var straceparse1_endre = regexp.MustCompile(`^(?P<pid>\d+)\s+\<\.\.\.\s+(?P<func>[^?][^ ]+) resumed\> (?P<body>.*)$`)
var straceparse1_usere = regexp.MustCompile(`^(?P<pid>\d+)\s+(?P<func>[^(]+)(?P<body>\(.*\s+= ((-?[0-9]+)|(\?))( \(.*\))?)`)

type straceparse1_state struct {
	wait  bool
	mid   []string
	start string
}

// Parse a single line of strace; can render multiple lines from state.
func straceparse1_line(state *straceparse1_state, line string) <-chan string {
	if !straceparse1_basre.MatchString(line) {
		log.Fatal("strace1: could not match strace base in " + line)
	}
	c := make(chan string)
	go func() {
		defer close(c)
		if state.wait {
			if !straceparse1_endre.MatchString(line) {
				state.mid = append(state.mid, line)
			} else {
				m := re_findmap(straceparse1_endre, line)
				c <- state.start + m["body"]
				mid := state.mid
				state.wait = false
				state.mid = []string{}
				state.start = ""
				for _, l := range mid {
					for l2 := range straceparse1_line(state, l) {
						c <- l2
					}
				}
			}
		} else {
			switch {
			case straceparse1_inire.MatchString(line):
				m := re_findmap(straceparse1_inire, line)
				state.start = m["pid"] + m["ini"]
				state.wait = true
			case straceparse1_usere.MatchString(line):
				c <- line
			case straceparse1_sigre.MatchString(line):
				// do nothing, we cannot deal with signals
			default:
				// invalid line, skip
				//log.Fatal("strace1: no regexp matches line "+line)
			}
		}
	}()
	return c
}

// StraceParse1: strace level 1 parser that joins unfinished lines in proper
// order.
func StraceParse1(c <-chan string) <-chan string {
	d := make(chan string)
	go func() {
		defer close(d)
		state := straceparse1_state{false, []string{}, ""}
		for l := range c {
			for l2 := range straceparse1_line(&state, l) {
				d <- l2
			}
		}
	}()
	return d
}

// strace level 2 parser: ####################################################

var strace2_re = regexp.MustCompile(`(?P<pid>\d+)\s+(?P<syscall>[^(]+)\((?P<body>.*)\)\s+= (?P<result>((-?[0-9]+)|(\?)))( (?P<error>((([^ ]+) \(.*\))|<unavailable>)))?$`)

// Strace2Info is the structured strace output.
type Strace2Info struct {
	pid     int
	syscall string
	body    string
	result  int
	err     string
	args    []string
}

// StraceParse2_argsplit splits strace function arguments into a list.
func StraceParse2_argsplit(s string) []string {
	args := []string{}
	arg := []string{}
	seps := map[string]string{`"`: `"`, "{": "}", "[": "]"}
	inside := ""
	for _, a0 := range s {
		a := string(a0)
		arg = append(arg, a)
		l := len(arg)
		if inside == "" && l > 2 && arg[l-2] == "," && arg[l-1] == " " {
			args = append(args, strings.Join(arg[:l-2], ""))
			arg = []string{}
			continue
		}
		switch {
		case inside != "" && a == seps[inside] && !(arg[l-2] == `\` && arg[l-1] == `"`):
			inside = ""
		case inside == "" && seps[a] != "":
			inside = a
		}
	}
	args = append(args, strings.Join(arg, ""))
	return args
}

// StraceParse2: strace level 2 parser that interprets complete lines and
// returns the structured information.
func StraceParse2(c <-chan string) <-chan Strace2Info {
	d := make(chan Strace2Info)
	go func() {
		defer close(d)
		for l := range c {
			if !strace2_re.MatchString(l) {
				continue
			}
			m := re_findmap(strace2_re, l)
			pid, err := strconv.Atoi(m["pid"])
			if err != nil {
				log.Fatalf("could not convert \"%s\" to pid (%s)", m["pid"], err.Error())
			}
			result := 0
			if m["result"] != "?" {
				result, err = strconv.Atoi(m["result"])
				if err != nil {
					log.Fatal("could not convert to result: " + m["result"])
				}
			}
			info := Strace2Info{
				pid:     pid,
				body:    m["body"],
				syscall: m["syscall"],
				err:     m["error"],
				result:  result,
				args:    StraceParse2_argsplit(m["body"]),
			}
			d <- info
		}
	}()
	return d
}

// strace level 3 parser: ####################################################

// StraceParse3 uses the structured strace output to generate the
// files read/written information.
func StraceParse3(c <-chan Strace2Info) (map[string]bool, map[string]bool) {
	r := make(map[string]bool)
	w := make(map[string]bool)
	for i := range c {
		switch i.syscall {
		case "creat":
			w[i.args[0][1:len(i.args[0])-1]] = true
		case "open":
			if i.result != -1 {
				filename := i.args[0][1 : len(i.args[0])-1]
				args := i.args
				switch {
				case strings.Contains(args[1], "O_RDONLY"):
					r[filename] = true
				case strings.Contains(args[1], "O_WRONLY"):
					w[filename] = true
				default:
					log.Fatalf("unable to determine operation with %s", args[1])
				}
			}
		case "unlinkat":
			if i.result != -1 {
				filename := i.args[1][1 : len(i.args[1])-1]
				delete(r, filename)
				delete(w, filename)
			}
		}
	}
	return r, w
}

// top function: #############################################################

// FileTrace runs the given command and return two channels: the first with the
// files read and the second with the files written.
func FileTrace(command string, env []string, dir string) (map[string]bool, map[string]bool) {
	return StraceParse3(StraceParse2(StraceParse1(StraceRun(command, env, dir))))
}

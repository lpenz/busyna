package libbusyna

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// strace runner: ############################################################

// StraceRun runs a command using strace and writes the trace information to
// the returned channel.
// The environment variables and working directory can be specified.
// Returns a channel that receives the strace output, line-by-line.
func StraceRun(command string, env map[string]string, dir string) <-chan string {
	// Create a tmp file with the command, that removes itself at the end.
	cmdfile, err := ioutil.TempFile("", "busyna-strace-cmdfile-")
	cmdfile.WriteString(command)
	cmdfile.WriteString(fmt.Sprint("\n\n\nrm -f ", cmdfile.Name(), "\n"))

	// Create a tmp file for strace output.
	straceout, err := ioutil.TempFile("", "busyna-strace-output-")
	if err != nil {
		log.Fatal(err)
	}
	defer TmpEnd(straceout)

	// env2 will have the environment as expected by exec.Cmd
	var env2 []string
	if env == nil {
		env2 = nil
	} else {
		for k, v := range env {
			env2 = append(env2, fmt.Sprint(k, "=", v))
		}
	}

	// Build strace command-line and call it
	a := []string{
		"strace",
		"-f",
		"-a0",
		"-s1024",
		"-etrace=execve,clone,vfork,chdir,creat,rename,mkdir,rmdir,open",
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
		Env:    env2,
	}

	if err = cmd.Run(); err != nil {
		log.Fatal(err)
	}

	// Re-open the tmp strace output file to pass the second fd to the closure.
	// The first is closed by defer.
	return ChanFromFile(straceout.Name())
}

// strace level 1 parser: ####################################################

var straceparse1Basre = regexp.MustCompile(`^(?P<pid>\d+)\s+`)
var straceparse1Sigre = regexp.MustCompile(`^(?P<pid>\d+)\s+--- (?P<signal>[A-Z]+) \(([a-zA-Z ]+)\) @ [0-9]+ \([0-9]+\) ---$`)
var straceparse1Inire = regexp.MustCompile(`^(?P<pid>\d+)(?P<ini>\s+[^(]+\(.*) <unfinished ...>$`)
var straceparse1Endre = regexp.MustCompile(`^(?P<pid>\d+)\s+\<\.\.\.\s+(?P<func>[^?][^ ]+) resumed\> (?P<body>.*)$`)
var straceparse1Usere = regexp.MustCompile(`^(?P<pid>\d+)\s+(?P<func>[^(]+)(?P<body>\(.*\s+= ((-?[0-9]+)|(\?))( \(.*\))?)`)

type straceparse1State struct {
	wait  bool
	mid   []string
	start string
}

// Parse a single line of strace; can render multiple lines from state.
// Returns a channel that receives the joined lines (see StraceParse1).
func straceparse1Line(state *straceparse1State, line string) <-chan string {
	if !straceparse1Basre.MatchString(line) {
		log.Fatal("strace1: could not match strace base in " + line)
	}
	c := make(chan string)
	go func() {
		defer close(c)
		if state.wait {
			if !straceparse1Endre.MatchString(line) {
				state.mid = append(state.mid, line)
			} else {
				m := ReFindMap(straceparse1Endre, line)
				c <- state.start + m["body"]
				mid := state.mid
				state.wait = false
				state.mid = []string{}
				state.start = ""
				for _, l := range mid {
					for l2 := range straceparse1Line(state, l) {
						c <- l2
					}
				}
			}
		} else {
			switch {
			case straceparse1Inire.MatchString(line):
				m := ReFindMap(straceparse1Inire, line)
				state.start = m["pid"] + m["ini"]
				state.wait = true
			case straceparse1Usere.MatchString(line):
				c <- line
			case straceparse1Sigre.MatchString(line):
				// do nothing, we cannot deal with signals
			default:
				// invalid line, skip
				//log.Fatal("strace1: no regexp matches line "+line)
			}
		}
	}()
	return c
}

// StraceParse1 is the parser that joins unfinished lines in proper order.
// Takes a channel that outputs strace lines, as returned by StraceRun.
// Returns a channel that receives the joined lines.
func StraceParse1(straceChan <-chan string) <-chan string {
	rChan := make(chan string)
	go func() {
		defer close(rChan)
		state := straceparse1State{false, []string{}, ""}
		for l := range straceChan {
			for l2 := range straceparse1Line(&state, l) {
				rChan <- l2
			}
		}
	}()
	return rChan
}

// strace level 2 parser: ####################################################

var strace2Re = regexp.MustCompile(`(?P<pid>\d+)\s+(?P<syscall>[^(]+)\((?P<body>.*)\)\s+= (?P<result>((-?[0-9]+)|(\?)))( (?P<error>((([^ ]+) \(.*\))|<unavailable>)))?$`)

// Strace2Info is the structured strace output.
type Strace2Info struct {
	pid     int
	syscall string
	body    string
	result  int
	err     string
	args    []string
}

// StraceParse2Argsplit splits strace function arguments into a list.
func StraceParse2Argsplit(straceArgs string) []string {
	args := []string{}
	arg := []string{}
	seps := map[string]string{`"`: `"`, "{": "}", "[": "]"}
	inside := ""
	for _, a0 := range straceArgs {
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

// StraceParse2 is the parser that interprets complete lines and returns the
// structured information.
// Takes a channel that outputs strace joined lines, as returned by StraceParser1.
func StraceParse2(strace1Chan <-chan string) <-chan Strace2Info {
	rChan := make(chan Strace2Info)
	go func() {
		defer close(rChan)
		for l := range strace1Chan {
			if !strace2Re.MatchString(l) {
				continue
			}
			m := ReFindMap(strace2Re, l)
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
				args:    StraceParse2Argsplit(m["body"]),
			}
			rChan <- info
		}
	}()
	return rChan
}

// strace level 3 parser: ####################################################

// StraceParse3 uses the structured strace output to generate the
// files read/written information.
// Returns read files, written files.
func StraceParse3(siChan <-chan Strace2Info) (map[string]bool, map[string]bool) {
	r := make(map[string]bool)
	w := make(map[string]bool)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dir := wd
	for i := range siChan {
		if i.result < 0 {
			continue
		}
		switch i.syscall {
		case "creat":
			w[i.args[0][1:len(i.args[0])-1]] = true
		case "open":
			filename := i.args[0][1 : len(i.args[0])-1]
			if !path.IsAbs(filename) {
				filename, _ = filepath.Rel(wd, path.Join(dir, filename))
			}
			args := i.args
			switch {
			case strings.Contains(args[1], "O_RDONLY"):
				r[filename] = true
			case strings.Contains(args[1], "O_WRONLY"):
				w[filename] = true
			case strings.Contains(args[1], "O_RDWR"):
				w[filename] = true
				r[filename] = true
			default:
				log.Fatalf("unable to determine operation with %s", args[1])
			}
		case "unlinkat":
			filename := i.args[1][1 : len(i.args[1])-1]
			delete(r, filename)
			delete(w, filename)
		case "chdir":
			dir = i.args[0][1 : len(i.args[0])-1]
		}
	}
	return r, w
}

// top function: #############################################################

// FileTrace runs the given command and return two channels: the first with the
// files read and the second with the files written.
func FileTrace(command string, env map[string]string, dir string) (map[string]bool, map[string]bool) {
	r, w := StraceParse3(StraceParse2(StraceParse1(StraceRun(command, env, dir))))
	/* Failsafe: remove references to files that are no longer there */
	for f := range r {
		if st, err := os.Stat(f); os.IsNotExist(err) || !st.Mode().IsRegular() {
			delete(r, f)
		}
	}
	for f := range w {
		if st, err := os.Stat(f); os.IsNotExist(err) || !st.Mode().IsRegular() {
			delete(w, f)
		}
	}
	return r, w
}

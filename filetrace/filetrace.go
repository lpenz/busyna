// Package filetrace runs a command and traces the files read and written.
package filetrace

import (
	"bufio"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
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
	if err != nil {
		log.Fatal(err)
	}
	defer tmpend(cmdfile)

	cmdfile.WriteString(command)
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
		"-etrace=execve,clone,vfork,chdir,creat,unlink,rename,mkdir,rmdir,open",
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

var strace1_basre = regexp.MustCompile(`^(?P<pid>\d+)\s+`)
var strace1_sigre = regexp.MustCompile(`^(?P<pid>\d+)\s+--- (?P<signal>[A-Z]+) \(([a-zA-Z ]+)\) @ [0-9]+ \([0-9]+\) ---$`)
var strace1_inire = regexp.MustCompile(`^(?P<pid>\d+)(?P<ini>\s+[^(]+\(.*) <unfinished ...>$`)
var strace1_endre = regexp.MustCompile(`^(?P<pid>\d+)\s+\<\.\.\.\s+(?P<func>[^?][^ ]+) resumed\> (?P<body>.*)$`)
var strace1_usere = regexp.MustCompile(`^(?P<pid>\d+)\s+(?P<func>[^(]+)(?P<body>\(.*\s+= ((-?[0-9]+)|(\?))( \(.*\))?)`)

type strace1_state struct {
	wait  bool
	mid   []string
	start string
}

// Parse a single line of strace; can render multiple lines from state.
func straceparser1_line(state *strace1_state, line string) <-chan string {
	if !strace1_basre.MatchString(line) {
		log.Fatal("strace1: could not match strace base in " + line)
	}
	c := make(chan string)
	go func() {
		defer close(c)
		if state.wait {
			if !strace1_endre.MatchString(line) {
				state.mid = append(state.mid, line)
			} else {
				m := re_findmap(strace1_endre, line)
				c <- state.start + m["body"]
				mid := state.mid
				state.wait = false
				state.mid = []string{}
				state.start = ""
				for _, l := range mid {
					for l2 := range straceparser1_line(state, l) {
						c <- l2
					}
				}
			}
		} else {
			switch {
			case strace1_inire.MatchString(line):
				m := re_findmap(strace1_inire, line)
				state.start = m["pid"] + m["ini"]
				state.wait = true
			case strace1_usere.MatchString(line):
				c <- line
			case strace1_sigre.MatchString(line):
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
		state := strace1_state{false, []string{}, ""}
		for l := range c {
			for l2 := range straceparser1_line(&state, l) {
				d <- l2
			}
		}
	}()
	return d
}

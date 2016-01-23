package libbusyna

import (
	"errors"
	"path/filepath"
	"regexp"
)

// parser: ###################################################################

// Parser internal state.
type parseState struct {
	env      map[string]string
	dir      string
	prevdir  string
	filename string
	linenum  int
	err      error
}

var parseEmptyRe = regexp.MustCompile(`^\s*$`)
var parseCommentRe = regexp.MustCompile(`^\s*\(?\s*[#:].*`)
var parseChdirRe = regexp.MustCompile(`^\s*cd\s+(?P<dir>.+)\s*$`)
var parseEnvRe = regexp.MustCompile(`^\s*(?P<key>[a-zA-Z_][a-zA-Z0-9_]*)=(?P<val>[a-zA-Z0-9_]*)\s*$`)
var parseEnvEscapedRe = regexp.MustCompile(`^\s*(?P<key>[a-zA-Z_][a-zA-Z0-9_]*)=["'](?P<val>[^']*)["']\s*$`)
var parseUnenvRe = regexp.MustCompile(`^\s*unset\s+(?P<key>[a-zA-Z_][a-zA-Z0-9_]*)\s*$`)
var parseSetsRe = regexp.MustCompile(`^\s*set\s+.*`)

// envcopy copies the provided env to another map
func envcopy(env map[string]string) map[string]string {
	r := map[string]string{}
	for k, v := range env {
		r[k] = v
	}
	return r
}

// Parse a single line from a busynarc file.
// Returns the channel that receives the structured result of the parsing.
func parseLine(state *parseState, line string) <-chan Cmd {
	rChan := make(chan Cmd)
	go func() {
		defer close(rChan)
		state.linenum++
		switch {
		case parseChdirRe.MatchString(line):
			m := ReFindMap(parseChdirRe, line)
			prevdir := state.dir
			switch {
			case m["dir"] == "-":
				state.dir = state.prevdir
			case filepath.IsAbs(string(m["dir"][0])):
				state.dir = m["dir"]
			default:
				state.dir = filepath.Join(state.dir, m["dir"])
			}
			state.prevdir = prevdir
		case parseEnvRe.MatchString(line):
			m := ReFindMap(parseEnvRe, line)
			state.env[m["key"]] = m["val"]
		case parseEnvEscapedRe.MatchString(line):
			m := ReFindMap(parseEnvEscapedRe, line)
			state.env[m["key"]] = m["val"]
		case parseUnenvRe.MatchString(line):
			m := ReFindMap(parseUnenvRe, line)
			delete(state.env, m["key"])
		case parseSetsRe.MatchString(line):
			// skip shell set's
		case parseEmptyRe.MatchString(line):
			// skip empty lines
		case parseCommentRe.MatchString(line):
			// skip comments
		default:
			// command line
			cmd := Cmd{line, envcopy(state.env), state.dir, state.err}
			if state.err == nil && filepath.IsAbs(state.dir) {
				errstr := "busyna.rc should use only relative directories"
				cmd.Err = errors.New(errstr)
			}
			rChan <- cmd
			state.err = nil
		}
	}()
	return rChan
}

// Parse a busynarc by channel.
func RcParse(filename string, c <-chan string) <-chan Cmd {
	rChan := make(chan Cmd)
	go func() {
		defer close(rChan)
		state := parseState{map[string]string{}, ".", "", filename, 0, nil}
		for l := range c {
			for l2 := range parseLine(&state, l) {
				rChan <- l2
			}
		}
	}()
	return rChan
}

// Parse a busynarc file.
func RcParseFile(rcfilename string) <-chan Cmd {
	return RcParse(rcfilename, ChanFromFile(rcfilename))
}

// runner: ###################################################################

// Run the provided command, and output the dependencies and targets to the
// provided channel.
func rcRun1(cmd Cmd, o chan<- CmdData) {
	deps, targets := FileTrace(cmd.Line, cmd.Env, cmd.Dir)
	cmddata := CmdData{cmd, deps, targets}
	o <- cmddata
}

// Run all commands in channel, outputing the dependencies and targets to the
// second argument.
func RcRun(c <-chan Cmd) <-chan CmdData {
	rvChan := make(chan CmdData)
	go func() {
		defer close(rvChan)
		for l := range c {
			rcRun1(l, rvChan)
		}
	}()
	return rvChan
}

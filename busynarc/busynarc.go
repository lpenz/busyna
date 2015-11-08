// Package busynarc deals with busynarc files
package busynarc

import (
	"github.com/lpenz/busyna/misc"
	"regexp"
)

// A single shell command and the environment where it should be executed.
type Command struct {
	env map[string]string
	dir string
	cmd string
}

// parser: ###################################################################

// Parser internal state.
type parseState struct {
	env map[string]string
	dir string
}

var parseEmptyRe = regexp.MustCompile(`^\s*$`)
var parseCommentRe = regexp.MustCompile(`^\s*#.*`)
var parseChdirRe = regexp.MustCompile(`^\s*cd\s+(?P<dir>.+)\s*$`)
var parseEnvRe = regexp.MustCompile(`^\s*(?P<key>[a-zA-Z_][a-zA-Z0-9_]*)=(?P<val>[a-zA-Z0-9_]*)\s*$`)
var parseUnenvRe = regexp.MustCompile(`^\s*unset\s+(?P<key>[a-zA-Z_][a-zA-Z0-9_]*)\s*$`)

// Parse a single line.
func parseLine(state *parseState, line string) <-chan Command {
	c := make(chan Command)
	go func() {
		defer close(c)
		switch {
		case parseChdirRe.MatchString(line):
			m := misc.ReFindMap(parseChdirRe, line)
			state.dir = m["dir"]
		case parseEnvRe.MatchString(line):
			m := misc.ReFindMap(parseEnvRe, line)
			state.env[m["key"]] = m["val"]
		case parseUnenvRe.MatchString(line):
			m := misc.ReFindMap(parseUnenvRe, line)
			state.env[m["key"]] = m["val"]
		case parseEmptyRe.MatchString(line):
			// skip empty lines
		case parseCommentRe.MatchString(line):
			// skip comments
		default:
			// command line
			env := map[string]string{}
			for k, v := range state.env {
				env[k] = v
			}
			cmd := Command{env, state.dir, line}
			c <- cmd
		}
	}()
	return c
}

// Parse a busynarc by channel.
func Parse(c <-chan string) <-chan Command {
	d := make(chan Command)
	go func() {
		defer close(d)
		state := parseState{map[string]string{}, ""}
		for l := range c {
			for l2 := range parseLine(&state, l) {
				d <- l2
			}
		}
	}()
	return d
}

// Parse a busynarc file
func ParseFile(rcfilename string) <-chan Command {
	return Parse(misc.ChanFromFile(rcfilename))
}

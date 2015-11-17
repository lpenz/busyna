package busynarc

import (
	"regexp"

	"github.com/lpenz/busyna/misc"
)

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

// Parse a single line from a busynarc file.
// Returns the channel that receives the structured result of the parsing.
func parseLine(state *parseState, line string) <-chan Cmd {
	rChan := make(chan Cmd)
	go func() {
		defer close(rChan)
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
			cmd := Cmd{line, env, state.dir}
			rChan <- cmd
		}
	}()
	return rChan
}

// Parse a busynarc by channel.
func Parse(c <-chan string) <-chan Cmd {
	rChan := make(chan Cmd)
	go func() {
		defer close(rChan)
		state := parseState{map[string]string{}, ""}
		for l := range c {
			for l2 := range parseLine(&state, l) {
				rChan <- l2
			}
		}
	}()
	return rChan
}

// Parse a busynarc file.
func ParseFile(rcfilename string) <-chan Cmd {
	return Parse(misc.ChanFromFile(rcfilename))
}

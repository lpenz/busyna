package libbusyna

import (
	"reflect"
)

// A single shell command and the environment where it should be executed.
// It is a stripped-down os/exec Cmd structure
type Cmd struct {
	Line string
	Env  map[string]string
	Dir  string
	Err  error
}

// A shell command with the discovered dependencies and targets.
type CmdData struct {
	Cmd     Cmd
	Deps    map[string]bool
	Targets map[string]bool
}

// CmdEqual checks if the provided Cmd's are equal
func CmdEqual(cmd1 Cmd, cmd2 Cmd) bool {
	return reflect.DeepEqual(cmd1, cmd2)
}

// CmdDataEqual checks if the provided CmdData's are equal
// strict controls wheather the deps of the second can be a subset of the deps
// of the first.
func CmdDataEqual(cmddata1 CmdData, cmddata2 CmdData, strict bool) bool {
	if !CmdEqual(cmddata1.Cmd, cmddata2.Cmd) {
		return false
	}
	for i := range cmddata2.Targets {
		if _, ok := cmddata1.Targets[i]; !ok {
			return false
		}
	}
	for i := range cmddata1.Targets {
		if _, ok := cmddata2.Targets[i]; !ok {
			return false
		}
	}
	for i := range cmddata2.Deps {
		if _, ok := cmddata1.Deps[i]; !ok {
			return false
		}
	}
	if !strict {
		return true
	}
	for i := range cmddata1.Deps {
		if _, ok := cmddata2.Deps[i]; !ok {
			return false
		}
	}
	return true
}

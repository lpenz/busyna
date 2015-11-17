// Types related to busyna.rc files
package libbusyna

// A single shell command and the environment where it should be executed.
// It is a stripped-down os/exec Cmd structure
type Cmd struct {
	Line string
	Env  map[string]string
	Dir  string
}

// A shell command with the discovered dependencies and targets.
type CmdData struct {
	Cmd     Cmd
	Deps    map[string]bool
	Targets map[string]bool
}

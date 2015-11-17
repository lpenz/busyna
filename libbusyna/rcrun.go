// Runner of busyna.rc files
package libbusyna

// runner: ###################################################################

// Run the provided command, and output the dependencies and targets to the
// provided channel.
func Run1(cmd Cmd, o chan<- CmdData) {
	deps, targets := FileTrace(cmd.Line, cmd.Env, cmd.Dir)
	cmddata := CmdData{cmd, deps, targets}
	o <- cmddata
}

// Run all commands in channel, outputing the dependencies and targets to the
// second argument.
func Run(c <-chan Cmd) <-chan CmdData {
	rvChan := make(chan CmdData)
	go func() {
		defer close(rvChan)
		for l := range c {
			Run1(l, rvChan)
		}
	}()
	return rvChan
}

// Package busynarc deals with busynarc files
package busynarc

// A single shell command and the environment where it should be executed.
type Command struct {
	env map[string]string
	dir string
	cmd string
}

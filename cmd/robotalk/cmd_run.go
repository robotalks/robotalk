package main

import "github.com/robotalks/robotalk/engine"

// RunCommand implements robotalk run
type RunCommand struct {
	URL   string
	Quiet bool
	Spec  string
}

// Execute implements Executable
func (c *RunCommand) Execute(args []string) error {
	return engine.Run(c.URL, c.Spec)
}

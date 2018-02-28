package cli

import (
	"os"

	"github.com/robotalks/talk/core/engine"
	"github.com/robotalks/talk/core/plugin"
)

// RunCommand implements robotalk run
type RunCommand struct {
	URL         string
	ModulesDir  []string `n:"modules-dir"`
	LoadModules bool     `n:"load-modules"`
	Quiet       bool
	Spec        string
}

// Execute implements Executable
func (c *RunCommand) Execute(args []string) error {
	os.Setenv("MQHUB_URL", c.URL)
	if c.LoadModules {
		plugin.LoadModules(c.ModulesDir)
	}
	return engine.Run(c.URL, c.Spec)
}

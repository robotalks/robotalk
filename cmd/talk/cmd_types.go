package main

import (
	"os"

	"github.com/robotalks/talk/core/cli"
)

// TypesCommand implements robotalk types
type TypesCommand struct {
	ModulesDir  []string `n:"modules-dir"`
	LoadModules bool     `n:"load-modules"`
}

// Execute implements Executable
func (c *TypesCommand) Execute(args []string) error {
	if c.LoadModules {
		loadModules(c.ModulesDir)
	}
	cli.PrintTypes(os.Stdout)
	return nil
}

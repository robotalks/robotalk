package main

import (
	"fmt"
	"sort"

	"github.com/robotalks/robotalk/engine"
)

// TypesCommand implements robotalk types
type TypesCommand struct{}

// Execute implements Executable
func (c *TypesCommand) Execute(args []string) error {
	names := make([]string, 0, len(engine.InstanceTypes))
	for name := range engine.InstanceTypes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

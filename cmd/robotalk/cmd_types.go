package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/robotalks/robotalk/engine"
)

// TypesCommand implements robotalk types
type TypesCommand struct{}

// Execute implements Executable
func (c *TypesCommand) Execute(args []string) error {
	names := make([]string, 0, len(engine.InstanceTypes))
	maxlen := 0
	for name := range engine.InstanceTypes {
		names = append(names, name)
		if l := len(name); l > maxlen {
			maxlen = l
		}
	}
	sort.Strings(names)
	for _, name := range names {
		line := name
		for len(line) < maxlen {
			line += " "
		}
		descs := strings.Split(engine.InstanceTypes[name].Description(), "\n")
		fmt.Println(line + " " + descs[0])
		if len(descs) > 1 {
			indent := " "
			for i := 0; i < maxlen; i++ {
				indent += " "
			}
			for _, str := range descs[1:] {
				fmt.Println(indent + str)
			}
		}
	}
	return nil
}

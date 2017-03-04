package cli

import (
	"fmt"
	"sort"
	"strings"

	talk "github.com/robotalks/talk.contract/v0"
	"github.com/robotalks/talk/engine"
)

// TypesCommand implements robotalk types
type TypesCommand struct {
	ModulesDir []string `n:"modules-dir"`
}

// Execute implements Executable
func (c *TypesCommand) Execute(args []string) error {
	engine.LoadModules(c.ModulesDir)
	types := talk.DefaultComponentTypeRegistry.RegisteredComponentTypes()
	typesMap := make(map[string]talk.ComponentType)
	names := make([]string, 0, len(types))
	maxlen := 0
	for _, t := range types {
		name := t.Name()
		typesMap[name] = t
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
		descs := strings.Split(typesMap[name].Description(), "\n")
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
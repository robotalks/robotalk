// +build cgo

package main

import (
	"github.com/robotalks/talk/core/plugin"

	_ "github.com/robotalks/talk/components/builtin"
)

func loadModules(moduleDirs []string) {
	plugin.LoadModules(moduleDirs)
}

package main

import (
	"fmt"

	"github.com/codingbrain/clix.go/exts/bind"
	"github.com/codingbrain/clix.go/exts/help"
	"github.com/codingbrain/clix.go/flag"
	"github.com/codingbrain/clix.go/term"

	"github.com/robotalks/talk/cli"

	_ "github.com/robotalks/mqhub.go/mqtt"
	_ "github.com/robotalks/talk/builtin"
)

// Version number
const Version = "0.1.0"

// VersionSuffix provides suffix info
var VersionSuffix = "-dev"

// PrintVersion prints version
func PrintVersion() {
	fmt.Println(Version + VersionSuffix)
}

type versionCommand struct {
}

func (c *versionCommand) Execute(_ []string) error {
	PrintVersion()
	return nil
}

func main() {
	cmd := &flag.CliDef{
		Cli: &flag.Command{
			Name: "talk",
			Desc: "Connect Robotic Components",
			Options: []*flag.Option{
				&flag.Option{
					Name:    "url",
					Alias:   []string{"s"},
					Desc:    "mqhub URL",
					Default: "mqtt://127.0.0.1:1883",
				},
				&flag.Option{
					Name:  "modules-dir",
					Alias: []string{"L"},
					Desc:  "path to look for modules",
					List:  true,
				},
				&flag.Option{
					Name:    "load-modules",
					Desc:    "load modules",
					Type:    "bool",
					Default: true,
				},
				&flag.Option{
					Name:  "quiet",
					Alias: []string{"q"},
					Desc:  "Turn off the logs",
					Type:  "bool",
				},
			},
			Commands: []*flag.Command{
				&flag.Command{
					Name: "run",
					Desc: "Run Components",
					Arguments: []*flag.Option{
						&flag.Option{
							Name:     "spec",
							Desc:     "Components spec file",
							Required: true,
							Type:     "string",
							Tags:     map[string]interface{}{"help-var": "SPEC"},
						},
					},
				},
				&flag.Command{
					Name: "types",
					Desc: "List all known types",
				},
				&flag.Command{
					Name: "version",
					Desc: "Show version",
				},
			},
		},
	}
	cmd.Normalize()
	cmd.Use(term.NewExt()).
		Use(bind.NewExt().
			Bind(&cli.RunCommand{}, "run").
			Bind(&cli.TypesCommand{}, "types").
			Bind(&versionCommand{}, "version")).
		Use(help.NewExt()).
		Parse().
		Exec()
}

package shell

import (
	"os"
	"os/exec"

	"github.com/robotalks/talk/contract/v0"
	cmn "github.com/robotalks/talk/core/common"
	eng "github.com/robotalks/talk/core/engine"
)

// Component is the implementation
type Component struct {
	Command string   `map:"command"`
	Shell   []string `map:"shell"`
	WorkDir string   `map:"workdir"`

	ref v0.ComponentRef
	cmd *exec.Cmd
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{ref: ref}
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	if len(s.Shell) == 0 {
		s.Shell = []string{"/bin/sh", "-c"}
	}
	return s, nil
}

// Ref implements v0.Component
func (s *Component) Ref() v0.ComponentRef {
	return s.ref
}

// Type implements v0.Component
func (s *Component) Type() v0.ComponentType {
	return Type
}

// Start implements v0.LifecycleCtl
func (s *Component) Start() error {
	cmd := exec.Command(s.Shell[0], append(s.Shell[1:], s.Command)...)
	cmd.Dir = s.WorkDir
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err == nil {
		s.cmd = cmd
	}
	return err
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	return cmn.StopCmd(s.cmd)
}

// Type is the Component type
var Type = eng.DefineComponentType("shell",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[BuiltIn] Execute Shell Command").
	Register()

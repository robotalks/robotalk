package shell

import (
	"os"
	"os/exec"

	talk "github.com/robotalks/talk.contract/v0"
	cmn "github.com/robotalks/talk/common"
	eng "github.com/robotalks/talk/engine"
)

// Component is the implementation
type Component struct {
	Command string   `json:"command"`
	Shell   []string `json:"shell"`
	WorkDir string   `json:"workdir"`

	ref talk.ComponentRef
	cmd *exec.Cmd
}

// NewComponent creates an Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{ref: ref}
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	if len(s.Shell) == 0 {
		s.Shell = []string{"/bin/sh", "-c"}
	}
	return s, nil
}

// Ref implements talk.Component
func (s *Component) Ref() talk.ComponentRef {
	return s.ref
}

// Type implements talk.Component
func (s *Component) Type() talk.ComponentType {
	return Type
}

// Start implements talk.LifecycleCtl
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

// Stop implements talk.LifecycleCtl
func (s *Component) Stop() error {
	return cmn.StopCmd(s.cmd)
}

// Type is the Component type
var Type = eng.DefineComponentType("shell",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[BuiltIn] Execute Shell Command").
	Register()

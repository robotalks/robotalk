package cmd

import (
	"os"
	"os/exec"

	cmn "github.com/robotalks/robotalk/common"
	eng "github.com/robotalks/robotalk/engine"
)

// Instance is the implementation
type Instance struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	WorkDir string   `json:"workdir"`

	cmd *exec.Cmd
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{}
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Start implements LifecycleCtl
func (s *Instance) Start() error {
	cmd := exec.Command(s.Command, s.Args...)
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

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	return cmn.StopCmd(s.cmd)
}

// Type is the instance type
var Type = eng.DefineInstanceType("cmd",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[BuiltIn] Execute External Program").
	Register()

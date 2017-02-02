package shell

import (
	"os"
	"os/exec"

	cmn "github.com/robotalks/robotalk/builtin/common"
	eng "github.com/robotalks/robotalk/engine"
)

// Instance is the implementation
type Instance struct {
	Command string   `json:"command"`
	Shell   []string `json:"shell"`
	WorkDir string   `json:"workdir"`

	cmd *exec.Cmd
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{}
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	if len(s.Shell) == 0 {
		s.Shell = []string{"/bin/sh", "-c"}
	}
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Start implements LifecycleCtl
func (s *Instance) Start() error {
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

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	return cmn.StopCmd(s.cmd)
}

// Type is the instance type
var Type = eng.DefineInstanceType("shell",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[BuiltIn] Execute Shell Command").
	Register()

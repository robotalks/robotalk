package common

import (
	"os"
	"os/exec"
	"time"
)

// CmdStopTimeout defines the time to wait after signals the process to exit
var CmdStopTimeout = 5 * time.Second

// StopCmd gracefully stop a process, if not, kill it
func StopCmd(cmd *exec.Cmd) error {
	proc := cmd.Process
	proc.Signal(os.Interrupt)
	ch := make(chan error)
	go func() {
		_, err := proc.Wait()
		ch <- err
	}()
	select {
	case <-time.After(CmdStopTimeout):
	case <-ch:
	}
	proc.Kill()
	return cmd.Wait()
}

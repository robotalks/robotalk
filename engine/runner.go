package engine

import (
	"log"
	"os"
	"os/signal"

	"github.com/easeway/langx.go/errors"
	"github.com/robotalks/mqhub.go/mqhub"
)

// Runner is a simple wrapper to run the engines
type Runner struct {
	HubURL    string
	SpecFile  string
	Logger    *log.Logger
	Spec      *Spec
	Connector mqhub.Connector
}

// Start implements LifecycleCtl
func (r *Runner) Start() error {
	if r.Spec == nil {
		raw := NewMapConfig()
		if err := raw.LoadFile(r.SpecFile); err != nil {
			return err
		}
		spec, err := ParseSpec(raw)
		if err != nil {
			return err
		}
		spec.Logger = r.Logger
		if err = spec.Resolve(); err != nil {
			return err
		}
		r.Spec = spec
	}
	if r.Connector == nil {
		conn, err := mqhub.NewConnector(r.HubURL)
		if err != nil {
			return err
		}
		r.Connector = conn
	}
	if err := r.Connector.Connect().Wait(); err != nil {
		return err
	}
	return r.Spec.Connect(r.Connector)
}

// Stop implements LifecycleCtl
func (r *Runner) Stop() error {
	var errs errors.AggregatedError
	errs.Add(r.Spec.Disconnect())
	errs.Add(r.Connector.Close())
	return errs.Aggregate()
}

// Run runs the engine
func (r *Runner) Run() error {
	if err := r.Start(); err != nil {
		return err
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	return r.Stop()
}

// LogPrefix is default log prefix
var LogPrefix = "RoboTalk:> "

// Run is the simple wrapper to run the engine
func Run(hubURL, specFile string) error {
	runner := &Runner{
		HubURL:   hubURL,
		SpecFile: specFile,
		Logger:   log.New(os.Stderr, LogPrefix, log.LstdFlags),
	}
	return runner.Run()
}

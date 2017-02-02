package firmata

import (
	"time"

	"github.com/hybridgroup/gobot"
	plat "github.com/hybridgroup/gobot/platforms/firmata"
	cmn "github.com/robotalks/robotalk/board/gobot/common"
	eng "github.com/robotalks/robotalk/engine"
)

// Config defines firmata configuration
type Config struct {
	SerialPort string `json:"serialport"`
}

// Instance is the implement of firmata instance
type Instance struct {
	Config
	adaptor *plat.FirmataAdaptor
}

// NewInstance creates a new instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{}
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	s.adaptor = plat.NewFirmataAdaptor(spec.FullID(), s.SerialPort)
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Start implements LifecycleCtl
func (s *Instance) Start() error {
	return firmataConnect(s.adaptor)
}

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	return cmn.Errs(s.adaptor.Finalize())
}

// Adaptor implements cmn.Adapter
func (s *Instance) Adaptor() gobot.Adaptor {
	return s.adaptor
}

// HACK: the first time firmata connect after cold boot never gets back

func firmataConnect(adaptor *plat.FirmataAdaptor) error {
	connCh := make(chan error)
	go func() {
		connCh <- cmn.Errs(adaptor.Connect())
	}()
	select {
	case e := <-connCh:
		return e
	case <-time.After(time.Second):
		adaptor.Disconnect()
	}
	// now reconnect
	return cmn.Errs(adaptor.Connect())
}

// Type is the instance type
var Type = eng.DefineInstanceType("gobot.adapter.firmata",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[GoBot] Firmata Adapter").
	Register()

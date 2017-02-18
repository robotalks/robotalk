package firmata

import (
	"time"

	eng "github.com/robotalks/robotalk/engine"
	"gobot.io/x/gobot"
	plat "gobot.io/x/gobot/platforms/firmata"
)

// Config defines firmata configuration
type Config struct {
	SerialPort string `json:"serialport"`
}

// Instance is the implement of firmata instance
type Instance struct {
	Config
	adaptor *plat.Adaptor
}

// NewInstance creates a new instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{}
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	s.adaptor = plat.NewAdaptor(s.SerialPort)
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
	return s.adaptor.Finalize()
}

// Adaptor implements cmn.Adapter
func (s *Instance) Adaptor() gobot.Adaptor {
	return s.adaptor
}

// HACK: the first time firmata connect after cold boot never gets back

func firmataConnect(adaptor *plat.Adaptor) error {
	connCh := make(chan error)
	go func() {
		connCh <- adaptor.Connect()
	}()
	select {
	case e := <-connCh:
		return e
	case <-time.After(time.Second):
		adaptor.Disconnect()
	}
	// now reconnect
	return adaptor.Connect()
}

// Type is the instance type
var Type = eng.DefineInstanceType("gobot.adapter.firmata",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[GoBot] Firmata Adapter").
	Register()

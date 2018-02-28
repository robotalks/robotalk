package firmata

import (
	"time"

	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
	"gobot.io/x/gobot"
	plat "gobot.io/x/gobot/platforms/firmata"
)

// Config defines firmata configuration
type Config struct {
	SerialPort string `map:"serialport"`
}

// Component is the implement of firmata Component
type Component struct {
	Config
	ref     v0.ComponentRef
	adaptor *plat.Adaptor
}

// NewComponent creates a new Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{ref: ref}
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	s.adaptor = plat.NewAdaptor(s.SerialPort)
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
	return firmataConnect(s.adaptor)
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	return s.adaptor.Finalize()
}

// Adaptor implements cmn.Adapter
func (s *Component) Adaptor() gobot.Adaptor {
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

// Type is the Component type
var Type = eng.DefineComponentType("gobot.adapter.firmata",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] Firmata Adapter").
	Register()

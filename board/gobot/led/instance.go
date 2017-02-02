package led

import (
	"fmt"

	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/robotalk/board/gobot/common"
	eng "github.com/robotalks/robotalk/engine"
)

// Config defines led configuration
type Config struct {
	Pin string `json:"pin"`
}

// Instance is the implement of led instance
type Instance struct {
	Config
	Adapter cmn.Adapter `key:"gpio" json:"-"`

	device *gpio.LedDriver
	state  *mqhub.DataPoint
	power  *mqhub.Reactor
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{state: &mqhub.DataPoint{Name: "state", Retain: true}}
	s.power = mqhub.ReactorAs("power", s.SetPower)
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	digitalWriter, ok := s.Adapter.Adaptor().(gpio.DigitalWriter)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.DigitalWriter", spec.FullID())
	}
	s.device = gpio.NewLedDriver(digitalWriter, spec.FullID(), s.Pin)
	if err := cmn.Errs(s.device.Start()); err != nil {
		return nil, err
	}
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Endpoints implements Stateful
func (s *Instance) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.state, s.power}
}

// State presents LED state
type State struct {
	On         bool  `json:"on"`
	Brightness *byte `json:"brightness,omitempty"`
}

// SetPower sets the power state
func (s *Instance) SetPower(state State) {
	switch {
	case !state.On:
		s.device.Off()
	case state.Brightness != nil:
		s.device.Brightness(*state.Brightness)
	default:
		s.device.On()
	}
}

// Type is the instance type
var Type = eng.DefineInstanceType("gobot.gpio.led",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[GoBot] GPIO LED").
	Register()

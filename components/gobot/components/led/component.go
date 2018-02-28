package led

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk-gobot/common"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
	"gobot.io/x/gobot/drivers/gpio"
)

// Config defines led configuration
type Config struct {
	Pin string `map:"pin"`
}

// Component is the implement of led Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"gpio" map:"-"`

	ref    talk.ComponentRef
	device *gpio.LedDriver
	state  *mqhub.DataPoint
	power  *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{ref: ref, state: &mqhub.DataPoint{Name: "state", Retain: true}}
	s.power = mqhub.ReactorAs("power", s.SetPower)
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	digitalWriter, ok := s.Adapter.Adaptor().(gpio.DigitalWriter)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.DigitalWriter", ref.MessagePath())
	}
	s.device = gpio.NewLedDriver(digitalWriter, s.Pin)
	if err := s.device.Start(); err != nil {
		return nil, err
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

// Endpoints implements talk.Stateful
func (s *Component) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.state, s.power}
}

// State presents LED state
type State struct {
	On         bool  `json:"on"`
	Brightness *byte `json:"brightness,omitempty"`
}

// SetPower sets the power state
func (s *Component) SetPower(state State) {
	switch {
	case !state.On:
		s.device.Off()
	case state.Brightness != nil:
		s.device.Brightness(*state.Brightness)
	default:
		s.device.On()
	}
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.gpio.led",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] GPIO LED").
	Register()

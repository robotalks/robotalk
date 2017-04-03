package pin

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk-gobot/common"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
)

// Config defines led configuration
type Config struct {
	Pin            string `map:"pin"`
	InitialDigital *bool  `map:"value"`
	InitialPWM     *byte  `map:"value"`
}

// Component is the implement of led Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"gpio" map:"-"`

	ref          talk.ComponentRef
	device       *gpio.DirectPinDriver
	writeDigital *mqhub.Reactor
	writePWM     *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{ref: ref}
	s.writeDigital = mqhub.ReactorAs("on", s.WriteDigital)
	s.writePWM = mqhub.ReactorAs("pwm", s.WritePWM)
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	conn, ok := s.Adapter.Adaptor().(gobot.Connection)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.Connection", ref.MessagePath())
	}
	s.device = gpio.NewDirectPinDriver(conn, s.Pin)
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
	return []mqhub.Endpoint{s.writeDigital, s.writePWM}
}

// Start implements talk.LifecycleCtl
func (s *Component) Start() error {
	if d := s.InitialDigital; d != nil {
		s.WriteDigital(*d)
	}
	if pwm := s.InitialPWM; pwm != nil {
		s.WritePWM(*pwm)
	}
	return nil
}

// Stop implements talk.LifecycleCtl
func (s *Component) Stop() error {
	return nil
}

// WriteDigital writes digital 0/1 to pin
func (s *Component) WriteDigital(v bool) {
	if v {
		s.device.On()
	} else {
		s.device.Off()
	}
}

// WritePWM writes to PWM pin
func (s *Component) WritePWM(pwm byte) {
	s.device.PwmWrite(pwm)
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.gpio.pin",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] GPIO Pin").
	Register()

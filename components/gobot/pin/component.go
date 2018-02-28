package pin

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk/components/gobot/common"
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
)

// Config defines pin configuration
type Config struct {
	Pin            string `map:"pin"`
	InitialDigital *bool  `map:"value"`
	InitialPWM     *byte  `map:"value"`
	StopDigital    *bool  `map:"stop-value"`
	StopPWM        *byte  `map:"stop-value"`
}

// Component is the implement of pin Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"gpio" map:"-"`

	ref          v0.ComponentRef
	device       *gpio.DirectPinDriver
	writeDigital *mqhub.Reactor
	writePWM     *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
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

// Ref implements v0.Component
func (s *Component) Ref() v0.ComponentRef {
	return s.ref
}

// Type implements v0.Component
func (s *Component) Type() v0.ComponentType {
	return Type
}

// Endpoints implements v0.Stateful
func (s *Component) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.writeDigital, s.writePWM}
}

// Start implements v0.LifecycleCtl
func (s *Component) Start() error {
	if d := s.InitialDigital; d != nil {
		s.WriteDigital(*d)
	}
	if pwm := s.InitialPWM; pwm != nil {
		s.WritePWM(*pwm)
	}
	return nil
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	if d := s.StopDigital; d != nil {
		s.WriteDigital(*d)
	}
	if pwm := s.StopPWM; pwm != nil {
		s.WritePWM(*pwm)
	}
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
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] GPIO Pin").
	Register()

package pwm

import (
	"fmt"
	"log"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk/components/gobot/common"
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
)

// Config defines servo configuration
type Config struct {
	Channel    int     `map:"channel"`
	PulseMin   int     `map:"pulse-min"`
	PulseMax   int     `map:"pulse-max"`
	InitialPos float32 `map:"initial-pos"`
	Reverse    bool    `map:"reverse"`
}

// State defines the state of this component
type State struct {
	Pos   float32 `json:"pos"`
	Pulse uint    `json:"pulse"`
}

// Component is the implement of PWM driven servo
type Component struct {
	Config
	Driver cmn.PWMDriver `inject:"pwm" map:"-"`

	ref   v0.ComponentRef
	state *mqhub.DataPoint
	pos   *mqhub.Reactor
	pulse *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{
		Config: Config{
			PulseMin: 100,
			PulseMax: 1000,
		},
		ref:   ref,
		state: &mqhub.DataPoint{Name: "state", Retain: true},
	}
	s.pos = mqhub.ReactorAs("pos", s.SetServoPos)
	s.pulse = mqhub.ReactorAs("pulse", s.setPulse)

	if err := eng.SetupComponent(s, ref); err != nil {
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
	return []mqhub.Endpoint{s.state, s.pos, s.pulse}
}

// Start implements v0.LifecycleCtl
func (s *Component) Start() error {
	return s.SetServoPos(s.InitialPos)
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	return nil
}

// SetServoPos implements cmn.Servo
func (s *Component) SetServoPos(pos float32) error {
	if pos < -1.0 || pos > 1.0 {
		return fmt.Errorf("invalid pos %f", pos)
	}
	if s.Reverse {
		pos = -pos
	}
	pulse := uint(s.PulseMin + int((pos+1.0)*float32(s.PulseMax-s.PulseMin)/2.0))
	if err := s.Driver.SetPWMPulse(s.Channel, 0, pulse); err != nil {
		log.Printf("[%s] SetPosition(%f)[chn=%d, pulse=%d] err: %v",
			s.ref.ComponentID(), pos, s.Channel, pulse, err)
		return err
	}

	s.updateState(pos, pulse)
	return nil
}

// for debug purpose only
func (s *Component) setPulse(value int) {
	pulse := uint(value)
	if err := s.Driver.SetPWMPulse(s.Channel, 0, pulse); err != nil {
		log.Printf("[%s] SetPulse(%d)[chn=%d] err: %v",
			s.ref.ComponentID(), pulse, s.Channel, err)
		return
	}
	pos := float32(int(pulse)-s.PulseMin)*2.0/float32(s.PulseMax-s.PulseMin) - 1.0
	s.updateState(pos, pulse)
}

func (s *Component) updateState(pos float32, pulse uint) {
	if s.Reverse {
		pos = -pos
	}
	s.state.Update(&State{Pos: pos, Pulse: pulse})
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.servo.pwm",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] PWM Servo").
	Register()

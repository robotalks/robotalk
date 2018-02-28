package motor

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk/components/gobot/common"
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
	"gobot.io/x/gobot/drivers/gpio"
)

// Config defines motor configuration
type Config struct {
	Pin     string  `map:"pin"`
	DirPin  string  `map:"dir-pin"`
	Mode    *string `map:"mode"`
	Reverse bool    `map:"reverse"`
}

// Component is the implement of motor Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"gpio" map:"-"`

	ref    v0.ComponentRef
	device *gpio.MotorDriver
	state  *mqhub.DataPoint
	speed  *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{ref: ref, state: &mqhub.DataPoint{Name: "state", Retain: true}}
	s.speed = mqhub.ReactorAs("speed", s.SetSpeed)
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	digitalWriter, ok := s.Adapter.Adaptor().(gpio.DigitalWriter)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.DigitalWriter", ref.MessagePath())
	}
	s.device = gpio.NewMotorDriver(digitalWriter, s.Pin)
	s.device.DirectionPin = s.DirPin
	s.device.CurrentMode = "analog"
	if s.Mode != nil {
		s.device.CurrentMode = *s.Mode
	}
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
	return []mqhub.Endpoint{s.state, s.speed}
}

// SetSpeed set the motor speed, speed is -1.0 ~ 1.0
func (s *Component) SetSpeed(speed float32) {
	if speed < -1.0 || speed > 1.0 {
		return
	}
	speedVal := int(255 * speed)
	if s.Reverse {
		speedVal = -speedVal
	}
	var err error
	switch {
	case speedVal == 0:
		err = s.device.Off()
	case speedVal < 0:
		err = s.device.Backward(byte(-speedVal))
	default:
		err = s.device.Forward(byte(speedVal))
	}
	if err == nil {
		s.state.Update(speed)
	}
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.gpio.motor",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] GPIO Motor").
	Register()

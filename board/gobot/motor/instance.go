package motor

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/robotalk/board/gobot/common"
	eng "github.com/robotalks/robotalk/engine"
	"gobot.io/x/gobot/drivers/gpio"
)

// Config defines motor configuration
type Config struct {
	Pin     string  `json:"pin"`
	DirPin  string  `json:"dir-pin"`
	Mode    *string `json:"mode"`
	Reverse bool    `json:"reverse"`
}

// Instance is the implement of motor instance
type Instance struct {
	Config
	Adapter cmn.Adapter `key:"gpio" json:"-"`

	device *gpio.MotorDriver
	state  *mqhub.DataPoint
	speed  *mqhub.Reactor
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{state: &mqhub.DataPoint{Name: "state", Retain: true}}
	s.speed = mqhub.ReactorAs("speed", s.SetSpeed)
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	digitalWriter, ok := s.Adapter.Adaptor().(gpio.DigitalWriter)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.DigitalWriter", spec.FullID())
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

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Endpoints implements Stateful
func (s *Instance) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.state, s.speed}
}

// SetSpeed set the motor speed, speed is -1.0 ~ 1.0
func (s *Instance) SetSpeed(speed float32) {
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

// Type is the instance type
var Type = eng.DefineInstanceType("gobot.gpio.motor",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[GoBot] GPIO Motor").
	Register()

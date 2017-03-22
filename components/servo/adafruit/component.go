package adafruit

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk-gobot/common"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
	"gobot.io/x/gobot/drivers/i2c"
)

// Config defines servo configuration
type Config struct {
	Channel    int     `map:"channel"`
	Freq       int     `map:"frequency"`
	PulseMin   int     `map:"pulse-min"`
	PulseMax   int     `map:"pulse-max"`
	InitialPos float32 `map:"initial-pos"`
}

// Component is the implement of Adafruit HAT Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"i2c" map:"-"`

	ref    talk.ComponentRef
	device *i2c.AdafruitMotorHatDriver
	state  *mqhub.DataPoint
	pos    *mqhub.Reactor
	pulse  *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{
		Config: Config{
			Freq:     60,
			PulseMin: 100,
			PulseMax: 1000,
		},
		ref:   ref,
		state: &mqhub.DataPoint{Name: "state", Retain: true},
	}
	s.pos = mqhub.ReactorAs("pos", s.SetPosition)
	s.pulse = mqhub.ReactorAs("pulse", s.setPulse)

	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}

	bus, ok := s.Adapter.Adaptor().(i2c.Connector)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not i2c", ref.MessagePath())
	}
	s.device = i2c.NewAdafruitMotorHatDriver(bus)
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
	return []mqhub.Endpoint{s.state, s.pos, s.pulse}
}

// Start implements talk.LifecycleCtl
func (s *Component) Start() (err error) {
	err = s.device.Start()
	if err == nil {
		err = s.device.SetServoMotorFreq(float64(s.Freq))
	}
	if err == nil {
		s.SetPosition(s.InitialPos)
	}
	return
}

// Stop implements talk.LifecycleCtl
func (s *Component) Stop() error {
	return s.device.Halt()
}

// SetPosition sets absolute position from -1.0 ~ 1.0
func (s *Component) SetPosition(pos float32) {
	if pos < -1.0 || pos > 1.0 {
		return
	}
	pulse := s.PulseMin + int((pos+1.0)*float32(s.PulseMax-s.PulseMin)/2.0)
	err := s.device.SetServoMotorPulse(byte(s.Channel), 0, int32(pulse))
	if err == nil {
		s.state.Update(pos)
	}
}

// for debug purpose only
func (s *Component) setPulse(pulse int) {
	s.device.SetServoMotorPulse(byte(s.Channel), 0, int32(pulse))
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.i2c.servo.adafruit",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] Adafruit Servo HAT (I2C)").
	Register()

package adafruit

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/robotalk/board/gobot/common"
	eng "github.com/robotalks/robotalk/engine"
	"gobot.io/x/gobot/drivers/i2c"
)

// Config defines servo configuration
type Config struct {
	Channel    int     `json:"channel"`
	Freq       int     `json:"frequency"`
	PulseMin   int     `json:"pulse-min"`
	PulseMax   int     `json:"pulse-max"`
	InitialPos float32 `json:"initial-pos"`
}

// Instance is the implement of Adafruit HAT instance
type Instance struct {
	Config
	Adapter cmn.Adapter `key:"i2c" json:"-"`

	device *i2c.AdafruitMotorHatDriver
	state  *mqhub.DataPoint
	pos    *mqhub.Reactor
	pulse  *mqhub.Reactor
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{
		Config: Config{
			Freq:     60,
			PulseMin: 100,
			PulseMax: 1000,
		},
		state: &mqhub.DataPoint{Name: "state", Retain: true},
	}
	s.pos = mqhub.ReactorAs("pos", s.SetPosition)
	s.pulse = mqhub.ReactorAs("pulse", s.setPulse)

	if err := spec.Reflect(s); err != nil {
		return nil, err
	}

	bus, ok := s.Adapter.Adaptor().(i2c.Connector)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not i2c", spec.FullID())
	}
	s.device = i2c.NewAdafruitMotorHatDriver(bus)
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Endpoints implements Stateful
func (s *Instance) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.state, s.pos, s.pulse}
}

// Start implements LifecycleCtl
func (s *Instance) Start() (err error) {
	err = s.device.Start()
	if err == nil {
		err = s.device.SetServoMotorFreq(float64(s.Freq))
	}
	if err == nil {
		s.SetPosition(s.InitialPos)
	}
	return
}

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	return s.device.Halt()
}

// SetPosition sets absolute position from -1.0 ~ 1.0
func (s *Instance) SetPosition(pos float32) {
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
func (s *Instance) setPulse(pulse int) {
	s.device.SetServoMotorPulse(byte(s.Channel), 0, int32(pulse))
}

// Type is the instance type
var Type = eng.DefineInstanceType("gobot.i2c.servo.adafruit",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[GoBot] Adafruit Servo HAT (I2C)").
	Register()

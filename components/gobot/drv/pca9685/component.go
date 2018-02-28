package pca9685

import (
	"fmt"
	"log"
	"time"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk/components/gobot/common"
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
	"gobot.io/x/gobot/drivers/i2c"
)

// Config defines servo configuration
type Config struct {
	Freq uint `map:"frequency"`
	Wait int  `map:"wait"`
}

// State defines the state of this component
type State struct {
	Freq uint `json:"freq"`
}

// Component is the implement of PCA9685 Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"i2c" map:"-"`

	ref    v0.ComponentRef
	device *i2c.PCA9685Driver
	state  *mqhub.DataPoint
	freq   *mqhub.Reactor
	pulse  *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{
		Config: Config{Freq: 50, Wait: 0},
		ref:    ref,
		state:  &mqhub.DataPoint{Name: "state", Retain: true},
	}
	s.freq = mqhub.ReactorAs("freq", s.SetPWMFrequency)
	s.pulse = mqhub.ReactorAs("pulse", s.setPulse)

	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}

	bus, ok := s.Adapter.Adaptor().(i2c.Connector)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not i2c", ref.MessagePath())
	}
	s.device = i2c.NewPCA9685Driver(bus)
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
	return []mqhub.Endpoint{s.state, s.freq, s.pulse}
}

// Start implements v0.LifecycleCtl
func (s *Component) Start() (err error) {
	if err = s.device.Start(); err != nil {
		return
	}
	return s.SetPWMFrequency(s.Freq)
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	return s.device.Halt()
}

// SetPWMFrequency implements common.PWMDriver
func (s *Component) SetPWMFrequency(freq uint) error {
	if err := s.device.SetPWMFreq(float32(freq)); err != nil {
		log.Printf("[%s] SetPWMFrequency(%d) err: %v", s.ref.ComponentID(), freq, err)
		return err
	}
	if s.Wait > 0 {
		time.Sleep(time.Duration(s.Wait) * time.Millisecond)
	}
	s.state.Update(&State{Freq: freq})
	return nil
}

// SetPWMPulse implements common.PWMDriver
func (s *Component) SetPWMPulse(chn int, on uint, off uint) error {
	return s.device.SetPWM(chn, uint16(on), uint16(off))
}

type setPulseParams struct {
	Ch  int  `json:"ch"`
	On  uint `json:"on"`
	Off uint `json:"off"`
}

func (s *Component) setPulse(params *setPulseParams) {
	if err := s.SetPWMPulse(params.Ch, params.On, params.Off); err != nil {
		log.Printf("[%s] SetPulse(%d, %d, %d) err: %v",
			s.ref.ComponentID(), params.Ch, params.On, params.Off, err)
	}
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.drv.pca9685",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] PCA9685 Driver (I2C)").
	Register()

package pca9685

import (
	"fmt"
	"log"
	"time"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk-gobot/common"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
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

	ref    talk.ComponentRef
	device *i2c.PCA9685Driver
	state  *mqhub.DataPoint
	freq   *mqhub.Reactor
	pulse  *mqhub.Reactor
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
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
	return []mqhub.Endpoint{s.state, s.freq, s.pulse}
}

// Start implements talk.LifecycleCtl
func (s *Component) Start() (err error) {
	if err = s.device.Start(); err != nil {
		return
	}
	return s.SetPWMFrequency(s.Freq)
}

// Stop implements talk.LifecycleCtl
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
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] PCA9685 Driver (I2C)").
	Register()

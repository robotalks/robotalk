package analog

import (
	"fmt"
	"time"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk-gobot/common"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
	"gobot.io/x/gobot/drivers/aio"
)

// Config defines analog sensor configuration
type Config struct {
	Pin      string `map:"pin"`
	Div      int    `map:"div"`
	Interval int    `map:"interval"`
}

// Component is the implement of analog sensor Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"io" map:"-"`

	ref        talk.ComponentRef
	device     *aio.AnalogSensorDriver
	state      *mqhub.DataPoint
	prevReport *int
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{
		Config: Config{Interval: 1000},
		ref:    ref,
		state:  &mqhub.DataPoint{Name: "value", Retain: true},
	}
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	conn, ok := s.Adapter.Adaptor().(aio.AnalogReader)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.aio.AnalogReader", ref.MessagePath())
	}
	s.device = aio.NewAnalogSensorDriver(conn, s.Pin, time.Duration(s.Interval)*time.Millisecond)
	s.device.On(aio.Data, func(v interface{}) {
		if val, ok := v.(int); ok {
			s.report(val)
		}
	})
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
	return []mqhub.Endpoint{s.state}
}

// Start implements talk.LifecycleCtl
func (s *Component) Start() error {
	val, err := s.device.Read()
	if err == nil {
		s.report(val)
	}
	return s.device.Start()
}

// Stop implements talk.LifecycleCtl
func (s *Component) Stop() error {
	return s.device.Halt()
}

func (s *Component) report(val int) {
	if s.Div > 0 {
		val /= s.Div
	}
	if s.prevReport == nil || val != *s.prevReport {
		s.prevReport = &val
		s.state.Update(val)
	}
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.analog",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] Analog Sensor").
	Register()

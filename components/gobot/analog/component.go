package analog

import (
	"fmt"
	"time"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk/components/gobot/common"
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
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

	ref        v0.ComponentRef
	device     *aio.AnalogSensorDriver
	state      *mqhub.DataPoint
	prevReport *int
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
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
	return []mqhub.Endpoint{s.state}
}

// Start implements v0.LifecycleCtl
func (s *Component) Start() error {
	val, err := s.device.Read()
	if err == nil {
		s.report(val)
	}
	return s.device.Start()
}

// Stop implements v0.LifecycleCtl
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
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] Analog Sensor").
	Register()

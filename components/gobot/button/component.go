package button

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk/components/gobot/common"
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
)

// Config defines button configuration
type Config struct {
	Pin     string `map:"pin"`
	Reverse bool   `map:"reverse"`
}

// Component is the implement of button Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"gpio" map:"-"`

	ref     v0.ComponentRef
	device  *gpio.ButtonDriver
	state   *mqhub.DataPoint
	eventCh chan *gobot.Event
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{ref: ref, state: &mqhub.DataPoint{Name: "state", Retain: true}}
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	digitalReader, ok := s.Adapter.Adaptor().(gpio.DigitalReader)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.DigitalReader", ref.MessagePath())
	}
	s.device = gpio.NewButtonDriver(digitalReader, s.Pin)
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
func (s *Component) Start() (err error) {
	if err = s.device.Start(); err != nil {
		return
	}
	s.eventCh = s.device.Subscribe()
	go s.run(s.eventCh)
	return
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	close(s.eventCh)
	return s.device.Halt()
}

func (s *Component) run(eventCh chan *gobot.Event) {
	for {
		event, ok := <-eventCh
		if !ok {
			break
		}
		if val, ok := event.Data.(int); ok {
			if val != 0 {
				val = 1
			}
			if s.Reverse {
				val = 1 - val
			}
			s.state.Update(val)
		}
	}
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.gpio.button",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] GPIO Button").
	Register()

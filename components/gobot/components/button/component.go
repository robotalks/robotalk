package button

import (
	"fmt"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk-gobot/common"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
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

	ref     talk.ComponentRef
	device  *gpio.ButtonDriver
	state   *mqhub.DataPoint
	eventCh chan *gobot.Event
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
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
func (s *Component) Start() (err error) {
	if err = s.device.Start(); err != nil {
		return
	}
	s.eventCh = s.device.Subscribe()
	go s.run(s.eventCh)
	return
}

// Stop implements talk.LifecycleCtl
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
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] GPIO Button").
	Register()

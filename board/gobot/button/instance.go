package button

import (
	"fmt"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/robotalk/board/gobot/common"
	eng "github.com/robotalks/robotalk/engine"
)

// Config defines button configuration
type Config struct {
	Pin     string `json:"pin"`
	Reverse bool   `json:"reverse"`
}

// Instance is the implement of button instance
type Instance struct {
	Config
	Adapter cmn.Adapter `key:"gpio" json:"-"`

	device  *gpio.ButtonDriver
	state   *mqhub.DataPoint
	eventCh chan *gobot.Event
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{state: &mqhub.DataPoint{Name: "state", Retain: true}}
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	digitalReader, ok := s.Adapter.Adaptor().(gpio.DigitalReader)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gobot.DigitalReader", spec.FullID())
	}
	s.device = gpio.NewButtonDriver(digitalReader, spec.FullID(), s.Pin)
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Endpoints implements Stateful
func (s *Instance) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.state}
}

// Start implements LifecycleCtl
func (s *Instance) Start() (err error) {
	if err = cmn.Errs(s.device.Start()); err != nil {
		return
	}
	s.eventCh = s.device.Subscribe()
	go s.run(s.eventCh)
	return
}

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	close(s.eventCh)
	return cmn.Errs(s.device.Halt())
}

func (s *Instance) run(eventCh chan *gobot.Event) {
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

// Type is the instance type
var Type = eng.DefineInstanceTypeAndRegister("gobot.gpio.button",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	}))

package raspi

import (
	"github.com/hybridgroup/gobot"
	plat "github.com/hybridgroup/gobot/platforms/raspi"
	eng "github.com/robotalks/robotalk/engine"
)

// Instance is the implement of Raspberry Pi instance
type Instance struct {
	adaptor *plat.RaspiAdaptor
}

// NewInstance creates a new instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{adaptor: plat.NewRaspiAdaptor(spec.FullID())}
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Adaptor implements cmn.Adapter
func (s *Instance) Adaptor() gobot.Adaptor {
	return s.adaptor
}

// Type is the instance type
var Type = eng.DefineInstanceType("gobot.adapter.raspi",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[GoBot] Raspberry Pi Adapter").
	Register()

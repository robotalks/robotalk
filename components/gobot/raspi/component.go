package raspi

import (
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
	"gobot.io/x/gobot"
	plat "gobot.io/x/gobot/platforms/raspi"
)

// Component is the implement of Raspberry Pi Component
type Component struct {
	ref     v0.ComponentRef
	adaptor *plat.Adaptor
}

// NewComponent creates a new Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{ref: ref, adaptor: plat.NewAdaptor()}
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

// Adaptor implements cmn.Adapter
func (s *Component) Adaptor() gobot.Adaptor {
	return s.adaptor
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.adapter.raspi",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] Raspberry Pi Adapter").
	Register()

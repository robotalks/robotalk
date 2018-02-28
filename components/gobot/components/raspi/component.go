package raspi

import (
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
	"gobot.io/x/gobot"
	plat "gobot.io/x/gobot/platforms/raspi"
)

// Component is the implement of Raspberry Pi Component
type Component struct {
	ref     talk.ComponentRef
	adaptor *plat.Adaptor
}

// NewComponent creates a new Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{ref: ref, adaptor: plat.NewAdaptor()}
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

// Adaptor implements cmn.Adapter
func (s *Component) Adaptor() gobot.Adaptor {
	return s.adaptor
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.adapter.raspi",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] Raspberry Pi Adapter").
	Register()

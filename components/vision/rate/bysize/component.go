package bysize

import (
	"sort"

	"github.com/robotalks/mqhub.go/mqhub"
	"github.com/robotalks/talk/components/vision/utils"
	"github.com/robotalks/talk/contract/v0"
	eng "github.com/robotalks/talk/core/engine"
)

// Component is the implementation
type Component struct {
	Objects mqhub.EndpointRef `inject:"objects" map:"-"`

	ref     v0.ComponentRef
	watcher mqhub.Watcher
	pub     *mqhub.DataPoint
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{
		ref: ref,
		pub: &mqhub.DataPoint{Name: "objects"},
	}
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
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
	return []mqhub.Endpoint{s.pub}
}

// Start implements v0.LifecycleCtl
func (s *Component) Start() (err error) {
	s.watcher, err = s.Objects.Watch(mqhub.MessageSinkAs(s.rateObjects))
	return
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	s.watcher.Close()
	return nil
}

func (s *Component) rateObjects(res *utils.Result) {
	sz := float32(res.Size.Square())
	for _, o := range res.Objects {
		rate := float32(o.Range.Square()) / sz
		o.Rate = &rate
	}
	sort.Sort(sort.Reverse(utils.ByRate(res.Objects)))
	s.pub.Update(res)
}

// Type is the component type
var Type = eng.DefineComponentType("vision.rate.bysize",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[Vision] Sort Detected Object By Size").
	Register()

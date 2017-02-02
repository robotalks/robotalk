package bysize

import (
	"sort"

	"github.com/robotalks/mqhub.go/mqhub"
	eng "github.com/robotalks/robotalk/engine"
	"github.com/robotalks/robotalk/logics/vision/utils"
)

// Instance is the implement of button instance
type Instance struct {
	Objects mqhub.EndpointRef `key:"objects" json:"-"`

	watcher mqhub.Watcher
	pub     *mqhub.DataPoint
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{pub: &mqhub.DataPoint{Name: "objects"}}
	if err := spec.Reflect(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Endpoints implements Stateful
func (s *Instance) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.pub}
}

// Start implements LifecycleCtl
func (s *Instance) Start() (err error) {
	s.watcher, err = s.Objects.Watch(mqhub.MessageSinkAs(s.rateObjects))
	return
}

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	s.watcher.Close()
	return nil
}

func (s *Instance) rateObjects(res *utils.Result) {
	sz := float32(res.Size.Square())
	for _, o := range res.Objects {
		rate := float32(o.Range.Square()) / sz
		o.Rate = &rate
	}
	sort.Sort(sort.Reverse(utils.ByRate(res.Objects)))
	s.pub.Update(res)
}

// Type is the instance type
var Type = eng.DefineInstanceType("vision.rate.bysize",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[Vision] Sort Detected Object By Size").
	Register()

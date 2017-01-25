package engine

import (
	"fmt"
	"path"
	"strings"

	"github.com/easeway/langx.go/errors"
	"github.com/robotalks/mqhub.go/mqhub"
)

// Spec is the top-level document
type Spec struct {
	Name        string                    `json:"name"`
	Version     string                    `json:"version"`
	Description string                    `json:"description"`
	Author      string                    `json:"author"`
	Children    map[string]*ComponentSpec `json:"components"`

	FactoryResolver InstanceFactoryResolver `json:"-"`

	initOrder   [][]*ComponentSpec
	publication mqhub.Publication
}

// ComponentSpec defines a specific component
type ComponentSpec struct {
	Factory     string                    `json:"factory"`
	Injections  map[string]string         `json:"inject"`
	Connections map[string]string         `json:"connect"`
	Children    map[string]*ComponentSpec `json:"components"`
	Config      map[string]interface{}    `json:"config"`

	LocalID            string                       `json:"-"`
	Root               *Spec                        `json:"-"`
	Parent             *ComponentSpec               `json:"-"`
	ResolvedFactory    InstanceFactory              `json:"-"`
	Instance           Instance                     `json:"-"`
	ResolvedInjections map[string]*ComponentSpec    `json:"-"`
	ConnectionRefs     map[string]mqhub.EndpointRef `json:"-"`

	depends   map[string]*ComponentSpec
	activates map[string]*ComponentSpec

	started bool
}

// ParseSpec parses spec from a config
func ParseSpec(input Config) (*Spec, error) {
	var spec Spec
	err := input.As(&spec)
	if err == nil && spec.Children != nil {
		for id, comp := range spec.Children {
			comp.init(&spec, id, nil)
		}
	}
	return &spec, err
}

// ID implements mqhub.Identifier
func (s *Spec) ID() string {
	return s.Name
}

// Endpoints implements mqhub.Component
func (s *Spec) Endpoints() []mqhub.Endpoint {
	return nil
}

// Components implements mqhub.Composite
func (s *Spec) Components() (comps []mqhub.Component) {
	for _, comp := range s.Children {
		comps = append(comps, comp)
	}
	return
}

// Resolve constructs the component instances
func (s *Spec) Resolve() error {
	all := make(map[string]*ComponentSpec)
	for _, comp := range s.Children {
		comp.resolveStart(all)
	}
	errs := &errors.AggregatedError{}
	for _, comp := range s.Children {
		comp.buildDependencies(errs)
	}
	if err := errs.Aggregate(); err != nil {
		return err
	}

	resolver := s.FactoryResolver
	if resolver == nil {
		resolver = DefaultInstanceFactoryResolver
	}
	for _, comp := range s.Children {
		comp.resolveFactory(resolver, errs)
	}
	if err := errs.Aggregate(); err != nil {
		return err
	}

	for len(all) > 0 {
		comps := make([]*ComponentSpec, 0, len(all))
		for _, comp := range all {
			comps = append(comps, comp)
		}
		ready := make([]*ComponentSpec, 0, len(all))
		for _, comp := range comps {
			ready = append(ready, comp.activate(all)...)
		}
		if len(ready) == 0 {
			for _, comp := range comps {
				errs.Add(fmt.Errorf("cyclic injection in %s", comp.FullID()))
			}
			break
		}
		s.initOrder = append(s.initOrder, ready)
	}
	return errs.Aggregate()
}

// Connect constructs all the component and establish endpoint connections
func (s *Spec) Connect(connector mqhub.Connector) error {
	errs := &errors.AggregatedError{}
	for _, comp := range s.Children {
		comp.resolveConnections(connector, errs)
	}
	if err := errs.Aggregate(); err != nil {
		return err
	}
	for _, group := range s.initOrder {
		for _, comp := range group {
			comp.connect(errs)
		}
	}
	if err := errs.Aggregate(); err != nil {
		return err
	}

	pub, err := connector.Publish(s)
	if err == nil {
		s.publication = pub
	}
	return err
}

// Disconnect tears off the components from mqhub
func (s *Spec) Disconnect() error {
	errs := &errors.AggregatedError{}
	for i := len(s.initOrder); i > 0; i-- {
		group := s.initOrder[i-1]
		for j := len(group); j > 0; j-- {
			errs.Add(group[j].disconnect())
		}
	}
	pub := s.publication
	s.publication = nil
	if pub != nil {
		errs.Add(pub.Close())
	}
	return errs.Aggregate()
}

// ID implements mqhub.Identifier
func (s *ComponentSpec) ID() string {
	return s.LocalID
}

// Endpoints implements mqhub.Component
func (s *ComponentSpec) Endpoints() []mqhub.Endpoint {
	if s.Instance != nil {
		return s.Instance.Endpoints()
	}
	return nil
}

// Components implements mqhub.Composite
func (s *ComponentSpec) Components() (comps []mqhub.Component) {
	for _, comp := range s.Children {
		comps = append(comps, comp)
	}
	return
}

// FullID returns the absolute ID reflecting the hierarchy
func (s *ComponentSpec) FullID() (id string) {
	for spec := s; spec != nil; spec = spec.Parent {
		if id != "" {
			id = spec.LocalID + "/" + id
		} else {
			id = spec.LocalID
		}
	}
	return
}

// ConfigAs maps config into provided type
func (s *ComponentSpec) ConfigAs(out interface{}) error {
	conf := &MapConfig{Map: s.Config}
	return conf.As(out)
}

func (s *ComponentSpec) init(root *Spec, id string, parent *ComponentSpec) {
	s.LocalID = id
	s.Root = root
	s.Parent = parent
	if s.Injections == nil {
		s.Injections = make(map[string]string)
	}
	if s.Connections == nil {
		s.Connections = make(map[string]string)
	}
	if s.Config == nil {
		s.Config = make(map[string]interface{})
	}
	if s.Children == nil {
		s.Children = make(map[string]*ComponentSpec)
	}
	for id, comp := range s.Children {
		comp.init(root, id, s)
	}
}

func (s *ComponentSpec) resolveStart(all map[string]*ComponentSpec) {
	all[s.FullID()] = s
	s.depends = make(map[string]*ComponentSpec)
	s.activates = make(map[string]*ComponentSpec)
	for _, comp := range s.Children {
		comp.resolveStart(all)
	}
}

func (s *ComponentSpec) buildDependencies(errs *errors.AggregatedError) {
	s.ResolvedInjections = make(map[string]*ComponentSpec)
	for name, injectID := range s.Injections {
		spec := s.resolveIDRef(injectID)
		if spec == nil {
			errs.Add(fmt.Errorf("%s: unresolved inject %s", s.FullID(), injectID))
			continue
		}
		s.depends[spec.FullID()] = spec
		spec.activates[s.FullID()] = s
		s.ResolvedInjections[name] = spec
	}
	for _, comp := range s.Children {
		// parent take dependency on child
		s.depends[comp.FullID()] = comp
		comp.activates[s.FullID()] = s
		comp.buildDependencies(errs)
	}
}

func (s *ComponentSpec) resolveIDRef(idRef string) *ComponentSpec {
	spec := s
	components := spec.Children
	if strings.HasPrefix(idRef, "/") {
		spec = nil
		components = s.Root.Children
		idRef = idRef[1:]
	}
	for idRef != "" {
		pos := strings.Index(idRef, "/")
		var id string
		if pos >= 0 {
			id = idRef[:pos]
			idRef = idRef[pos+1:]
		} else {
			id = idRef
			idRef = ""
		}
		if id == ".." {
			if spec == nil {
				return nil
			}
			spec = spec.Parent
		} else {
			spec = components[id]
		}
		if spec == nil {
			break
		}
		components = spec.Children
	}
	return spec
}

func (s *ComponentSpec) activate(all map[string]*ComponentSpec) (ready []*ComponentSpec) {
	if len(s.depends) == 0 && all[s.FullID()] != nil {
		ready = append(ready, s)
		delete(all, s.FullID())
		for _, spec := range s.activates {
			delete(spec.depends, s.FullID())
			ready = append(ready, spec.activate(all)...)
		}
	}
	for _, comp := range s.Children {
		ready = append(ready, comp.activate(all)...)
	}
	return
}

func (s *ComponentSpec) resolveFactory(resolver InstanceFactoryResolver, errs *errors.AggregatedError) {
	factory, err := resolver.ResolveInstanceFactory(s.Factory)
	errs.Add(err)
	if err == nil && factory == nil {
		errs.Add(fmt.Errorf("%s factory unresolved: %s", s.FullID(), s.Factory))
	}
	s.ResolvedFactory = factory
	for _, comp := range s.Children {
		comp.resolveFactory(resolver, errs)
	}
}

func (s *ComponentSpec) resolveConnections(connector mqhub.Connector, errs *errors.AggregatedError) {
	s.ConnectionRefs = make(map[string]mqhub.EndpointRef)
	for name, ref := range s.Connections {
		compRef, endpoint := path.Split(ref)
		s.ConnectionRefs[name] = connector.Describe(compRef).Endpoint(endpoint)
	}
	for _, comp := range s.Children {
		comp.resolveConnections(connector, errs)
	}
}

func (s *ComponentSpec) connect(errs *errors.AggregatedError) {
	instance, err := s.ResolvedFactory.CreateInstance(s)
	if !errs.Add(err) {
		s.Instance = instance
		if ctl, ok := instance.(LifecycleCtl); ok {
			s.started = !errs.Add(ctl.Start())
		}
	}
}

func (s *ComponentSpec) disconnect() error {
	inst := s.Instance
	s.Instance = nil
	if inst != nil && s.started {
		s.started = false
		if ctl, ok := inst.(LifecycleCtl); ok {
			return ctl.Stop()
		}
	}
	return nil
}

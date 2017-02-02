package engine

import (
	"fmt"
	"log"
	"path"
	"reflect"
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

	TypeResolver InstanceTypeResolver `json:"-"`
	Logger       *log.Logger          `json:"-"`

	initOrder   [][]*ComponentSpec
	publication mqhub.Publication
}

// ComponentSpec defines a specific component
type ComponentSpec struct {
	TypeName    string                    `json:"type"`
	Injections  map[string]string         `json:"inject"`
	Connections map[string]string         `json:"connect"`
	Children    map[string]*ComponentSpec `json:"components"`
	Config      map[string]interface{}    `json:"config"`

	LocalID            string                       `json:"-"`
	Root               *Spec                        `json:"-"`
	Parent             *ComponentSpec               `json:"-"`
	InstanceType       InstanceType                 `json:"-"`
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

	resolver := s.TypeResolver
	if resolver == nil {
		resolver = InstanceTypes
	}
	for _, comp := range s.Children {
		comp.resolveType(resolver, errs)
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
			errs.Add(group[j-1].disconnect())
		}
	}
	pub := s.publication
	s.publication = nil
	if pub != nil {
		errs.Add(pub.Close())
	}
	return errs.Aggregate()
}

// Logf wraps simple printf log
func (s *Spec) Logf(format string, v ...interface{}) {
	if l := s.Logger; l != nil {
		l.Printf(format, v...)
	}
}

// Logfln wraps simple printf log with newline
func (s *Spec) Logfln(format string, v ...interface{}) {
	if l := s.Logger; l != nil {
		l.Printf(format+"\n", v...)
	}
}

// ID implements mqhub.Identifier
func (s *ComponentSpec) ID() string {
	return s.LocalID
}

// Endpoints implements mqhub.Component
func (s *ComponentSpec) Endpoints() []mqhub.Endpoint {
	if s.Instance != nil {
		if stateful, ok := s.Instance.(Stateful); ok {
			return stateful.Endpoints()
		}
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

// Reflect maps config, injections, connections to dest struct
func (s *ComponentSpec) Reflect(dest interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(dest))
	if v.Kind() != reflect.Struct {
		panic("not a struct")
	}
	errs := errors.AggregatedError{}
	errs.Add(s.ConfigAs(dest))
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" || f.Anonymous { // unexported or anonymous field
			continue
		}
		if f.Type.Kind() != reflect.Interface {
			continue
		}

		key := f.Tag.Get("key")
		if key == "" {
			continue
		}

		// map connections
		if f.Type == reflect.TypeOf((*mqhub.EndpointRef)(nil)).Elem() {
			if ref, ok := s.ConnectionRefs[key]; ok {
				v.Field(i).Set(reflect.ValueOf(ref))
			} else {
				errs.Add(fmt.Errorf("%s: connection %s unspecified", s.FullID(), key))
			}
			continue
		}

		// otherwise, treat as injection
		comp := s.ResolvedInjections[key]
		if comp == nil || comp.Instance == nil {
			errs.Add(fmt.Errorf("%s: injection %s unspecified", s.FullID(), key))
			continue
		}
		instType := reflect.TypeOf(comp.Instance)
		fv := v.Field(i)
		if !fv.CanSet() {
			panic("field " + f.Name + " must be settable")
		}
		if instType.AssignableTo(f.Type) {
			v.Field(i).Set(reflect.ValueOf(comp.Instance))
		} else if instType.ConvertibleTo(f.Type) {
			v.Field(i).Set(reflect.ValueOf(comp.Instance).Convert(f.Type))
		} else {
			errs.Add(fmt.Errorf("%s injection %s type mismatch", s.FullID(), key))
		}
	}
	return errs.Aggregate()
}

// Logf wraps s.Root.Logf
func (s *ComponentSpec) Logf(format string, v ...interface{}) {
	s.Root.Logf(format, v...)
}

// Logfln wraps s.Root.Logfln
func (s *ComponentSpec) Logfln(format string, v ...interface{}) {
	s.Root.Logfln(format, v...)
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
	var components map[string]*ComponentSpec
	spec := s.Parent
	if spec != nil {
		components = spec.Children
	} else {
		components = s.Root.Children
	}
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
			if spec = spec.Parent; spec == nil {
				components = s.Root.Children
				continue
			}
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

func (s *ComponentSpec) resolveType(resolver InstanceTypeResolver, errs *errors.AggregatedError) {
	if s.TypeName == "" {
		s.InstanceType = nil
	} else {
		typ, err := resolver.ResolveInstanceType(s.TypeName)
		errs.Add(err)
		if err == nil && typ == nil {
			errs.Add(fmt.Errorf("%s type unresolved: %s", s.FullID(), s.TypeName))
		}
		s.InstanceType = typ
	}
	for _, comp := range s.Children {
		comp.resolveType(resolver, errs)
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
	if s.InstanceType == nil {
		return
	}
	s.Logfln("%s Initialize", s.FullID())
	instance, err := s.InstanceType.CreateInstance(s)
	if !errs.Add(err) {
		s.Instance = instance
		if ctl, ok := instance.(LifecycleCtl); ok {
			s.Logfln("%s Start", s.FullID())
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
			s.Logfln("%s Stop", s.FullID())
			return ctl.Stop()
		}
	}
	return nil
}

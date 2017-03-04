package engine

import (
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/easeway/langx.go/errors"
	"github.com/robotalks/mqhub.go/mqhub"
	talk "github.com/robotalks/talk.contract/v0"
)

// Spec is the top-level document
type Spec struct {
	Name        string                    `map:"name"`
	Version     string                    `map:"version"`
	Description string                    `map:"description"`
	Author      string                    `map:"author"`
	ChildSpecs  map[string]*ComponentSpec `map:"components"`

	TypeResolver talk.ComponentTypeResolver `map:"-"`
	Logger       *log.Logger                `map:"-"`

	initOrder   [][]*ComponentSpec
	publication mqhub.Publication
}

// Injection Types
const (
	InjectRef = "ref"
	InjectHub = "hub"
)

// InjectionSpec defines an injection
type InjectionSpec struct {
	Type string `map:"type"`
	ID   string `map:"id"`   // when type is ref
	Path string `map:"path"` // when type is hub
}

// ComponentSpec defines a specific component
type ComponentSpec struct {
	TypeName    string                    `map:"type"`
	InjectSpecs map[string]*InjectionSpec `map:"inject"`
	ChildSpecs  map[string]*ComponentSpec `map:"components"`
	Config      map[string]interface{}    `map:"config"`

	LocalID            string                 `map:"-"`
	Root               *Spec                  `map:"-"`
	ParentSpec         *ComponentSpec         `map:"-"`
	ResolvedType       talk.ComponentType     `map:"-"`
	Instance           talk.Component         `map:"-"`
	ResolvedInjections map[string]interface{} `map:"-"`

	depends   map[string]*ComponentSpec
	activates map[string]*ComponentSpec

	started bool
}

// ParseSpec parses spec from a config
func ParseSpec(input Config) (*Spec, error) {
	var spec Spec
	err := input.As(&spec)
	if err == nil && spec.ChildSpecs != nil {
		for id, s := range spec.ChildSpecs {
			s.init(&spec, id, nil)
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
	for _, spec := range s.ChildSpecs {
		comps = append(comps, spec)
	}
	return
}

// Resolve constructs the component instances
func (s *Spec) Resolve() error {
	all := make(map[string]*ComponentSpec)
	for _, spec := range s.ChildSpecs {
		spec.resolveStart(all)
	}
	errs := &errors.AggregatedError{}
	for _, spec := range s.ChildSpecs {
		spec.buildDependencies(errs)
	}
	if err := errs.Aggregate(); err != nil {
		return err
	}

	resolver := s.TypeResolver
	if resolver == nil {
		resolver = talk.DefaultComponentTypeRegistry
	}
	for _, spec := range s.ChildSpecs {
		spec.resolveType(resolver, errs)
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
	for _, spec := range s.ChildSpecs {
		spec.resolveConnections(connector, errs)
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
		if stateful, ok := s.Instance.(talk.Stateful); ok {
			return stateful.Endpoints()
		}
	}
	return nil
}

// Components implements mqhub.Composite
func (s *ComponentSpec) Components() (comps []mqhub.Component) {
	for _, spec := range s.ChildSpecs {
		comps = append(comps, spec)
	}
	return
}

// ComponentID implements talk.ComponentRef
func (s *ComponentSpec) ComponentID() string {
	return s.ID()
}

// MessagePath implements talk.ComponentRef
func (s *ComponentSpec) MessagePath() string {
	return s.FullID()
}

// ComponentConfig implements talk.ComponentRef
func (s *ComponentSpec) ComponentConfig() map[string]interface{} {
	return s.Config
}

// Injections implements talk.ComponentRef
func (s *ComponentSpec) Injections() map[string]interface{} {
	return s.ResolvedInjections
}

// Component implements talk.ComponentRef
func (s *ComponentSpec) Component() talk.Component {
	return s.Instance
}

// Parent implements talk.ComponentRef
func (s *ComponentSpec) Parent() talk.ComponentRef {
	return s.ParentSpec
}

// Children implements talk.ComponentRef
func (s *ComponentSpec) Children() []talk.ComponentRef {
	children := make([]talk.ComponentRef, 0, len(s.ChildSpecs))
	for _, spec := range s.ChildSpecs {
		children = append(children, spec)
	}
	return children
}

// FullID returns the absolute ID reflecting the hierarchy
func (s *ComponentSpec) FullID() (id string) {
	for spec := s; spec != nil; spec = spec.ParentSpec {
		if id != "" {
			id = spec.LocalID + "/" + id
		} else {
			id = spec.LocalID
		}
	}
	return
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
	s.ParentSpec = parent
	if s.InjectSpecs == nil {
		s.InjectSpecs = make(map[string]*InjectionSpec)
	}
	if s.Config == nil {
		s.Config = make(map[string]interface{})
	}
	if s.ChildSpecs == nil {
		s.ChildSpecs = make(map[string]*ComponentSpec)
	}
	for id, spec := range s.ChildSpecs {
		spec.init(root, id, s)
	}
}

func (s *ComponentSpec) resolveStart(all map[string]*ComponentSpec) {
	all[s.FullID()] = s
	s.depends = make(map[string]*ComponentSpec)
	s.activates = make(map[string]*ComponentSpec)
	for _, spec := range s.ChildSpecs {
		spec.resolveStart(all)
	}
}

func (s *ComponentSpec) buildDependencies(errs *errors.AggregatedError) {
	s.ResolvedInjections = make(map[string]interface{})
	for name, injectSpec := range s.InjectSpecs {
		if injectSpec.Type != InjectRef {
			continue
		}
		if injectSpec.ID == "" {
			errs.Add(fmt.Errorf("%s: injection 'id' required %s", s.FullID(), name))
		}
		spec := s.resolveIDRef(injectSpec.ID)
		if spec == nil {
			errs.Add(fmt.Errorf("%s: unresolved inject %s", s.FullID(), injectSpec.ID))
			continue
		}
		s.depends[spec.FullID()] = spec
		spec.activates[s.FullID()] = s
		s.ResolvedInjections[name] = spec
	}
	for _, spec := range s.ChildSpecs {
		// parent take dependency on child
		s.depends[spec.FullID()] = spec
		spec.activates[s.FullID()] = s
		spec.buildDependencies(errs)
	}
}

func (s *ComponentSpec) resolveIDRef(idRef string) *ComponentSpec {
	var components map[string]*ComponentSpec
	spec := s.ParentSpec
	if spec != nil {
		components = spec.ChildSpecs
	} else {
		components = s.Root.ChildSpecs
	}
	if strings.HasPrefix(idRef, "/") {
		spec = nil
		components = s.Root.ChildSpecs
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
			if spec = spec.ParentSpec; spec == nil {
				components = s.Root.ChildSpecs
				continue
			}
		} else {
			spec = components[id]
		}
		if spec == nil {
			break
		}
		components = spec.ChildSpecs
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
	for _, spec := range s.ChildSpecs {
		ready = append(ready, spec.activate(all)...)
	}
	return
}

func (s *ComponentSpec) resolveType(resolver talk.ComponentTypeResolver, errs *errors.AggregatedError) {
	if s.TypeName == "" {
		s.ResolvedType = nil
	} else {
		typ, err := resolver.ResolveComponentType(s.TypeName)
		errs.Add(err)
		if err == nil && typ == nil {
			errs.Add(fmt.Errorf("%s type unresolved: %s", s.FullID(), s.TypeName))
		}
		s.ResolvedType = typ
	}
	for _, spec := range s.ChildSpecs {
		spec.resolveType(resolver, errs)
	}
}

func (s *ComponentSpec) resolveConnections(connector mqhub.Connector, errs *errors.AggregatedError) {
	for name, inject := range s.InjectSpecs {
		if inject.Type != InjectHub || inject.Path == "" {
			continue
		}
		compRef, endpoint := path.Split(inject.Path)
		s.ResolvedInjections[name] = connector.Describe(compRef).Endpoint(endpoint)
	}
	for _, spec := range s.ChildSpecs {
		spec.resolveConnections(connector, errs)
	}
}

func (s *ComponentSpec) connect(errs *errors.AggregatedError) {
	if s.ResolvedType == nil {
		return
	}
	factory := s.ResolvedType.Factory()
	if factory == nil {
		return
	}
	s.Logfln("%s Initialize", s.FullID())
	instance, err := factory.CreateComponent(s)
	if !errs.Add(err) {
		s.Instance = instance
		if ctl, ok := instance.(talk.LifecycleCtl); ok {
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
		if ctl, ok := inst.(talk.LifecycleCtl); ok {
			s.Logfln("%s Stop", s.FullID())
			return ctl.Stop()
		}
	}
	return nil
}

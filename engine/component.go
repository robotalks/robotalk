package engine

import "github.com/robotalks/mqhub.go/mqhub"

// Instance is the instance of component logic
type Instance interface {
}

// Stateful defines instances which publishes endpoints
type Stateful interface {
	Endpoints() []mqhub.Endpoint
}

// LifecycleCtl provides start/stop control
type LifecycleCtl interface {
	Start() error
	Stop() error
}

// InstanceFactory creates instances
type InstanceFactory interface {
	// CreateInstance creates an instance
	CreateInstance(*ComponentSpec) (Instance, error)
}

// InstanceFactoryFunc is func form of InstanceFactory
type InstanceFactoryFunc func(spec *ComponentSpec) (Instance, error)

// CreateInstance implements InstanceFactory
func (f InstanceFactoryFunc) CreateInstance(spec *ComponentSpec) (Instance, error) {
	return f(spec)
}

// InstanceFactoryResolver resolves instance factory by name
type InstanceFactoryResolver interface {
	ResolveInstanceFactory(name string) (InstanceFactory, error)
}

// InstanceFactoryRegistry provides a registry for named instance factories
type InstanceFactoryRegistry struct {
	Factories map[string]InstanceFactory
}

// ResolveInstanceFactory implements InstanceFactoryResolver
func (r *InstanceFactoryRegistry) ResolveInstanceFactory(name string) (InstanceFactory, error) {
	return r.Factories[name], nil
}

// DefaultInstanceFactoryResolver provides the default implementation of
// InstanceFactoryResolver backed by InstanceFactoryRegistry
var DefaultInstanceFactoryResolver = &InstanceFactoryRegistry{Factories: make(map[string]InstanceFactory)}

// RegisterInstanceFactory registers a named instance factory
func RegisterInstanceFactory(name string, factory InstanceFactory) {
	DefaultInstanceFactoryResolver.Factories[name] = factory
}

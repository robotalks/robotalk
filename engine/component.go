package engine

import "github.com/robotalks/mqhub.go/mqhub"

// Instance is the instance of component logic
type Instance interface {
	Type() InstanceType
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

// InstanceType is the factory which creates instances
type InstanceType interface {
	// Name returns instance type name
	Name() string
	// CreateInstance creates an instance
	CreateInstance(*ComponentSpec) (Instance, error)
}

// InstanceTypeResolver resolves instance type by name
type InstanceTypeResolver interface {
	ResolveInstanceType(name string) (InstanceType, error)
}

// InstanceTypeRegistry provides a registry for named instance types
type InstanceTypeRegistry map[string]InstanceType

// ResolveInstanceType implements InstanceTypeResolver
func (r InstanceTypeRegistry) ResolveInstanceType(name string) (InstanceType, error) {
	return r[name], nil
}

// InstanceTypes is the default registry for all known instance types
var InstanceTypes = make(InstanceTypeRegistry)

// RegisterInstanceTypes registers named instance types
func RegisterInstanceTypes(types ...InstanceType) {
	for _, t := range types {
		InstanceTypes[t.Name()] = t
	}
}

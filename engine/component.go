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

// InstanceFactory creates instances
type InstanceFactory interface {
	// CreateInstance creates an instance
	CreateInstance(*ComponentSpec) (Instance, error)
}

// InstanceFactoryFunc is func form of InstanceFactory
type InstanceFactoryFunc func(*ComponentSpec) (Instance, error)

// CreateInstance implements InstanceFactory
func (f InstanceFactoryFunc) CreateInstance(spec *ComponentSpec) (Instance, error) {
	return f(spec)
}

// InstanceType is the factory which creates instances
type InstanceType interface {
	InstanceFactory
	// Name returns instance type name
	Name() string
	// Description returns detailed description of the type
	// CONVENTION: the first line is the summary
	Description() string
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

// CustomInstanceType is used as template to create instance types
type CustomInstanceType struct {
	TypeName string
	TypeDesc string
	Factory  InstanceFactory
}

// Name implements InstanceType
func (t *CustomInstanceType) Name() string {
	return t.TypeName
}

// Description implements InstanceType
func (t *CustomInstanceType) Description() string {
	return t.TypeDesc
}

// CreateInstance implements InstanceType
func (t *CustomInstanceType) CreateInstance(spec *ComponentSpec) (Instance, error) {
	return t.Factory.CreateInstance(spec)
}

// Describe provides type description
func (t *CustomInstanceType) Describe(desc string) *CustomInstanceType {
	t.TypeDesc = desc
	return t
}

// Register wraps RegisterInstanceType
func (t *CustomInstanceType) Register() *CustomInstanceType {
	RegisterInstanceTypes(t)
	return t
}

// DefineInstanceType defines a custom instance type
func DefineInstanceType(name string, factory InstanceFactory) *CustomInstanceType {
	return &CustomInstanceType{TypeName: name, Factory: factory}
}

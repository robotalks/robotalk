package v0

import "github.com/robotalks/mqhub.go/mqhub"

// Component is the abstract representative of an object in the topology
type Component interface {
	Ref() ComponentRef
	Type() ComponentType
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

// ComponentRef is a reference to the component
type ComponentRef interface {
	// ComponentID retrieves the ID of current component
	ComponentID() string
	// MessagePath is the path for dispatching message to this component
	MessagePath() string
	// ComponentConfig retrieves configuration for the component
	ComponentConfig() map[string]interface{}
	// Injections retrieves resolved injections
	Injections() map[string]interface{}
	// Component retrieves created component
	Component() Component
	// Parent retrieves parent component ref
	Parent() ComponentRef
	// Children retrieves child component refs
	Children() []ComponentRef
}

// ComponentFactory creates components
type ComponentFactory interface {
	// CreateComponent creates a component
	CreateComponent(ComponentRef) (Component, error)
}

// ComponentType is the type of components
type ComponentType interface {
	// Factory returns the ComponentFactory
	Factory() ComponentFactory
	// Name returns component type name
	Name() string
	// Description returns detailed description of the type
	// CONVENTION: the first line is the summary
	Description() string
}

// ComponentTypeResolver resolves instance type by name
type ComponentTypeResolver interface {
	ResolveComponentType(name string) (ComponentType, error)
}

// ComponentTypeRegistry registers component types
type ComponentTypeRegistry interface {
	ComponentTypeResolver
	RegisterComponentType(componentType ComponentType)
	RegisteredComponentTypes() []ComponentType
}

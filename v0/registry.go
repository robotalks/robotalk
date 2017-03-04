package talk

type mapRegistry map[string]ComponentType

func (r mapRegistry) ResolveComponentType(name string) (ComponentType, error) {
	return r[name], nil
}

func (r mapRegistry) RegisterComponentType(componentType ComponentType) {
	r[componentType.Name()] = componentType
}

func (r mapRegistry) RegisteredComponentTypes() (types []ComponentType) {
	types = make([]ComponentType, 0, len(r))
	for _, t := range r {
		types = append(types, t)
	}
	return
}

// DefaultComponentTypeRegistry is the default
var DefaultComponentTypeRegistry ComponentTypeRegistry = make(mapRegistry)

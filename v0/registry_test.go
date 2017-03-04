package talk

import (
	"testing"
)

type testComponentType struct {
	name  string
	value int
}

func (t *testComponentType) Factory() ComponentFactory { return nil }
func (t *testComponentType) Name() string              { return t.name }
func (t *testComponentType) Description() string       { return "" }

func TestRegisterComponentType(t *testing.T) {
	registry := make(mapRegistry)
	registry.RegisterComponentType(&testComponentType{name: "test0"})
	types := registry.RegisteredComponentTypes()
	if l := len(types); l != 1 {
		t.Errorf("expect 1 type, actual %d", l)
	} else if name := types[0].Name(); name != "test0" {
		t.Errorf("expect name to be test0, got %s", name)
	}
	resolved, err := registry.ResolveComponentType("test0")
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if resolved == nil {
		t.Errorf("expected type not found")
	} else if name := resolved.Name(); name != "test0" {
		t.Errorf("expect resolved type has name test0, got %s", name)
	}

	registry.RegisterComponentType(&testComponentType{name: "test1"})
	types = registry.RegisteredComponentTypes()
	if l := len(types); l != 2 {
		t.Errorf("expect 2 type, actual %d", l)
	}
}

func TestRegisterComponentTypeWithSameName(t *testing.T) {
	registry := make(mapRegistry)
	registry.RegisterComponentType(&testComponentType{name: "test0", value: 0})
	registry.RegisterComponentType(&testComponentType{name: "test0", value: 1})
	types := registry.RegisteredComponentTypes()
	if l := len(types); l != 1 {
		t.Errorf("expect 1 type, actual %d", l)
	} else if name := types[0].Name(); name != "test0" {
		t.Errorf("expect name to be test0, got %s", name)
	} else if v := types[0].(*testComponentType).value; v != 1 {
		t.Errorf("expect type.value is 1, got %d", v)
	}
}

func TestDefaultRegistry(t *testing.T) {
	if DefaultComponentTypeRegistry == nil {
		t.Error("DefaultComponentTypeRegistry is nil")
	}
}

package engine

import (
	"bytes"
	"path"
	"testing"

	"github.com/robotalks/mqhub.go/mqhub"
	talk "github.com/robotalks/talk.contract/v0"
	"github.com/stretchr/testify/assert"
)

type TestTypes map[string]talk.ComponentType

func (f TestTypes) ResolveComponentType(name string) (talk.ComponentType, error) {
	return f[name], nil
}

type tester struct {
	t      *testing.T
	assert *assert.Assertions
	types  TestTypes
}

func makeTester(t *testing.T) *tester {
	return &tester{
		t:      t,
		assert: assert.New(t),
		types:  make(TestTypes),
	}
}

// mock mqhub.Connector

type testDummyPublication struct {
	component mqhub.Component
}

func (p *testDummyPublication) Close() error {
	return nil
}

func (p *testDummyPublication) Component() mqhub.Component {
	return p.component
}

type testDummyDesc struct {
	componentID string
}

func (d *testDummyDesc) Watch(sink mqhub.MessageSink) (mqhub.Watcher, error) {
	// should not be called
	return nil, nil
}

func (d *testDummyDesc) ID() string {
	return d.componentID
}

func (d *testDummyDesc) SubComponent(id ...string) mqhub.Descriptor {
	return &testDummyDesc{componentID: path.Join(append([]string{d.componentID}, id...)...)}
}

func (d *testDummyDesc) Endpoint(name string) mqhub.EndpointRef {
	return &testDummyEndpointRef{componentID: d.componentID, endpoint: name}
}

type testDummyEndpointRef struct {
	componentID string
	endpoint    string
	messages    []mqhub.Message
}

func (r *testDummyEndpointRef) Watch(sink mqhub.MessageSink) (mqhub.Watcher, error) {
	// should not be called
	return nil, nil
}

func (r *testDummyEndpointRef) ConsumeMessage(msg mqhub.Message) mqhub.Future {
	r.messages = append(r.messages, msg)
	return &mqhub.ImmediateFuture{}
}

func (t *tester) Close() error {
	return nil
}

func (t *tester) Watch(sink mqhub.MessageSink) (mqhub.Watcher, error) {
	// TODO should not be called
	return nil, nil
}

func (t *tester) Connect() mqhub.Future {
	// TODO should not be called
	return nil
}

func (t *tester) Publish(comp mqhub.Component) (mqhub.Publication, error) {
	return &testDummyPublication{component: comp}, nil
}

func (t *tester) Describe(componentID string) mqhub.Descriptor {
	return &testDummyDesc{componentID: componentID}
}

func (t *tester) addTypes(types ...talk.ComponentType) {
	for _, typ := range types {
		t.types[typ.Name()] = typ
	}
}

func (t *tester) spec(content string) *Spec {
	conf := NewMapConfig()
	t.assert.NoError(conf.Load(bytes.NewBufferString(content)))
	spec, err := ParseSpec(conf)
	t.assert.NoError(err)
	spec.TypeResolver = t.types
	t.assert.NoError(spec.Resolve())
	t.assert.NoError(spec.Connect(t))
	return spec
}

type specTester struct {
	t      *testing.T
	assert *assert.Assertions
	spec   *Spec
}

func newSpecTester(t *testing.T, spec *Spec) *specTester {
	return &specTester{
		t:      t,
		assert: assert.New(t),
		spec:   spec,
	}
}

func (t *specTester) component(id string) *ComponentSpec {
	comp := t.spec.ChildSpecs[id]
	t.assert.NotNil(comp)
	return comp
}

type testInstanceAType struct{}

func (t *testInstanceAType) Name() string                   { return "test.A" }
func (t *testInstanceAType) Description() string            { return t.Name() }
func (t *testInstanceAType) Factory() talk.ComponentFactory { return t }
func (t *testInstanceAType) CreateComponent(ref talk.ComponentRef) (talk.Component, error) {
	inst := &testInstanceA{ref: ref}
	return inst, SetupComponent(inst, ref)
}

var typeInstanceA = &testInstanceAType{}

type testInstanceA struct {
	Param string `map:"param"`
	ref   talk.ComponentRef
}

func (a *testInstanceA) Ref() talk.ComponentRef   { return a.ref }
func (a *testInstanceA) Type() talk.ComponentType { return typeInstanceA }
func (a *testInstanceA) Start() error             { return nil }
func (a *testInstanceA) Stop() error              { return nil }

type testInstanceBType struct{}

func (t *testInstanceBType) Name() string                   { return "test.B" }
func (t *testInstanceBType) Description() string            { return t.Name() }
func (t *testInstanceBType) Factory() talk.ComponentFactory { return t }
func (t *testInstanceBType) CreateComponent(ref talk.ComponentRef) (talk.Component, error) {
	inst := &testInstanceB{ref: ref}
	return inst, SetupComponent(inst, ref)
}

var typeInstanceB = &testInstanceBType{}

type testInstanceB struct {
	Ctl    talk.LifecycleCtl `inject:"a" json:"-"`
	Remote mqhub.EndpointRef `inject:"ref" json:"-"`
	ref    talk.ComponentRef
}

func (b *testInstanceB) Ref() talk.ComponentRef   { return b.ref }
func (b *testInstanceB) Type() talk.ComponentType { return typeInstanceB }

func TestSimpleComponents(t *testing.T) {
	tester := makeTester(t)
	tester.addTypes(typeInstanceA, typeInstanceB)
	spec := tester.spec(`---
        name: test
        components:
          a:
            type: test.A
            config:
              param: test
          b:
            type: test.B
            inject:
              a:
                type: ref
                id: a
              ref:
                type: hub
                path: remote/component/endpoint
          l1:
           components:
             b0:
               type: test.B
               inject:
                 a:
                   type: ref
                   id: /a
                 ref:
                   type: hub
                   path: remote/component/endpoint
             b1:
               type: test.B
               inject:
                 a:
                   type: ref
                   id: ../a
                 ref:
                   type: hub
                   path: remote/component/endpoint
     `)
	specT := newSpecTester(t, spec)
	assert.Equal(t, "test", specT.component("a").Instance.(*testInstanceA).Param)
	b := specT.component("b").Instance.(*testInstanceB)
	assert.NotNil(t, b.Ctl)
	assert.NotNil(t, b.Remote)
	assert.NoError(t, spec.Disconnect())
}

type testOrderType struct {
	start []string
	stop  []string
}

func (t *testOrderType) Name() string                   { return "test.order" }
func (t *testOrderType) Description() string            { return t.Name() }
func (t *testOrderType) Factory() talk.ComponentFactory { return t }
func (t *testOrderType) CreateComponent(ref talk.ComponentRef) (talk.Component, error) {
	inst := &testOrder{typ: t, ref: ref}
	return inst, SetupComponent(inst, ref)
}

func (t *testOrderType) onStart(ref talk.ComponentRef) {
	t.start = append(t.start, ref.(*ComponentSpec).FullID())
}

func (t *testOrderType) onStop(ref talk.ComponentRef) {
	t.stop = append(t.stop, ref.(*ComponentSpec).FullID())
}

type testOrder struct {
	typ *testOrderType
	ref talk.ComponentRef
}

func (a *testOrder) Ref() talk.ComponentRef   { return a.ref }
func (a *testOrder) Type() talk.ComponentType { return a.typ }
func (a *testOrder) Start() error             { a.typ.onStart(a.ref); return nil }
func (a *testOrder) Stop() error              { a.typ.onStop(a.ref); return nil }

func TestLifecycleOrders(t *testing.T) {
	tester := makeTester(t)
	compType := &testOrderType{}
	tester.addTypes(compType)
	spec := tester.spec(`---
        name: test
        components:
          a:
            type: test.order
            after:
              - l1
          b:
            type: test.order
            after:
              - a
              - l1
          l1:
           components:
             a0:
               type: test.order
             a1:
               type: test.order
               after:
                 - a0
     `)
	assert.Equal(t, []string{"l1/a0", "l1/a1", "a", "b"}, compType.start)
	spec.Disconnect()
	assert.Equal(t, []string{"b", "a", "l1/a1", "l1/a0"}, compType.stop)
}

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

func (t *tester) addTypes(types ...InstanceType) {
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
	comp := t.spec.Children[id]
	t.assert.NotNil(comp)
	return comp
}

type testInstanceAType struct{}

func (t *testInstanceAType) Name() string        { return "test.A" }
func (t *testInstanceAType) Description() string { return t.Name() }
func (t *testInstanceAType) CreateInstance(spec *ComponentSpec) (Instance, error) {
	inst := &testInstanceA{}
	return inst, spec.Reflect(inst)
}

var typeInstanceA = &testInstanceAType{}

type testInstanceA struct {
	Param string `json:"param"`
}

func (a *testInstanceA) Type() InstanceType { return typeInstanceA }
func (a *testInstanceA) Start() error       { return nil }
func (a *testInstanceA) Stop() error        { return nil }

type testInstanceBType struct{}

func (t *testInstanceBType) Name() string        { return "test.B" }
func (t *testInstanceBType) Description() string { return t.Name() }
func (t *testInstanceBType) CreateInstance(spec *ComponentSpec) (Instance, error) {
	inst := &testInstanceB{}
	return inst, spec.Reflect(inst)
}

var typeInstanceB = &testInstanceBType{}

type testInstanceB struct {
	Ctl    LifecycleCtl      `key:"a" json:"-"`
	Remote mqhub.EndpointRef `key:"ref" json:"-"`
}

func (b *testInstanceB) Type() InstanceType { return typeInstanceB }

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
              a: a
            connect:
              ref: remote/component/endpoint
          l1:
           components:
             b0:
               type: test.B
               inject:
                 a: /a
               connect:
                 ref: remote/component/endpoint
             b1:
               type: test.B
               inject:
                 a: ../a
               connect:
                 ref: remote/component/endpoint
     `)
	specT := newSpecTester(t, spec)
	assert.Equal(t, "test", specT.component("a").Instance.(*testInstanceA).Param)
	b := specT.component("b").Instance.(*testInstanceB)
	assert.NotNil(t, b.Ctl)
	assert.NotNil(t, b.Remote)
	assert.NoError(t, spec.Disconnect())
}

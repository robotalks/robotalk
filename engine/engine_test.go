package engine

import (
	"bytes"
	"path"
	"testing"

	"github.com/robotalks/mqhub.go/mqhub"
	"github.com/stretchr/testify/assert"
)

type TestFactories map[string]InstanceFactory

func (f TestFactories) ResolveInstanceFactory(name string) (InstanceFactory, error) {
	return f[name], nil
}

type tester struct {
	t         *testing.T
	assert    *assert.Assertions
	factories TestFactories
}

func makeTester(t *testing.T) *tester {
	return &tester{
		t:         t,
		assert:    assert.New(t),
		factories: make(TestFactories),
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

func (t *tester) addFactory(name string, f InstanceFactory) {
	t.factories[name] = f
}

func (t *tester) spec(content string) *Spec {
	conf := NewMapConfig()
	t.assert.NoError(conf.Load(bytes.NewBufferString(content)))
	spec, err := ParseSpec(conf)
	t.assert.NoError(err)
	spec.FactoryResolver = t.factories
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

type testInstanceA struct {
	Param string `json:"param"`
}

func (a *testInstanceA) Start() error {
	return nil
}

func (a *testInstanceA) Stop() error {
	return nil
}

type testInstanceB struct {
	Ctl    LifecycleCtl      `key:"a" json:"-"`
	Remote mqhub.EndpointRef `key:"ref" json:"-"`
}

func TestSimpleComponents(t *testing.T) {
	tester := makeTester(t)
	tester.addFactory("a", InstanceFactoryFunc(func(spec *ComponentSpec) (Instance, error) {
		inst := &testInstanceA{}
		return inst, spec.Map(inst)
	}))
	tester.addFactory("b", InstanceFactoryFunc(func(spec *ComponentSpec) (Instance, error) {
		inst := &testInstanceB{}
		return inst, spec.Map(inst)
	}))
	spec := tester.spec(`---
        name: test
        components:
          a:
            factory: a
            config:
              param: test
          b:
            factory: b
            inject:
              a: a
            connect:
              ref: remote/component/endpoint
          l1:
           components:
             b0:
               factory: b
               inject:
                 a: /a
             b1:
               factory: b
               inject:
                 a: ../a
     `)
	specT := newSpecTester(t, spec)
	assert.Equal(t, "test", specT.component("a").Instance.(*testInstanceA).Param)
	b := specT.component("b").Instance.(*testInstanceB)
	assert.NotNil(t, b.Ctl)
	assert.NotNil(t, b.Remote)
	assert.NoError(t, spec.Disconnect())
}

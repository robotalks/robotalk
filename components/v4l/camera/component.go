package camera

import (
	"fmt"
	"io"
	"log"

	"github.com/robotalks/mqhub.go/mqhub"
	"github.com/robotalks/talk/contract/v0"
	cmn "github.com/robotalks/talk/core/common"
	eng "github.com/robotalks/talk/core/engine"
)

// Config defines camera configuration
type Config struct {
	Device  string            `map:"device"`
	Width   int               `map:"width"`
	Height  int               `map:"height"`
	Format  string            `map:"format"`
	Quality *int              `map:"quality"`
	Casts   map[string]string `map:"cast"`
}

// State defines camera state
type State struct {
	On     bool   `json:"on"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	FourCC FourCC `json:"fourcc"`
}

// Component is the implementation
type Component struct {
	ref      v0.ComponentRef
	config   Config
	settings Options
	stateDp  *mqhub.DataPoint
	recvDp   *mqhub.DataPoint
	imageDp  *mqhub.DataPoint
	onOff    *mqhub.Reactor
	castTo   *mqhub.Reactor
	udpCast  *cmn.UDPCast
	casts    []cmn.CastTarget
	stream   *Stream
}

// NewComponent creates a Component
func NewComponent(ref v0.ComponentRef) (v0.Component, error) {
	s := &Component{
		ref: ref,
		config: Config{
			Device: "/dev/video0",
			Width:  640,
			Height: 480,
			Format: FourCCMJPG.String(),
		},
		stateDp: &mqhub.DataPoint{Name: "state", Retain: true},
		recvDp:  &mqhub.DataPoint{Name: "receiver", Retain: true},
	}
	mapConf := &eng.MapConfig{Map: ref.ComponentConfig()}
	err := mapConf.As(&s.config)
	if err != nil {
		return nil, err
	}

	s.onOff = mqhub.ReactorAs("on", s.setOn)
	s.castTo = mqhub.ReactorAs("cast", s.setCastTo)

	s.settings.Device = s.config.Device
	s.settings.FourCC, err = ParseFourCC(s.config.Format)
	if err != nil {
		return nil, err
	}
	s.settings.Width, s.settings.Height = s.config.Width, s.config.Height
	s.settings.Quality = s.config.Quality

	s.udpCast = &cmn.UDPCast{}
	s.casts = []cmn.CastTarget{s.udpCast}

	for t, val := range s.config.Casts {
		switch t {
		case "udp":
			s.casts = append(s.casts, &cmn.UDPCast{Address: val})
		case "endpoint":
			s.imageDp = &mqhub.DataPoint{Name: val}
			s.casts = append(s.casts, &cmn.DataPointCast{DP: s.imageDp})
		default:
			return nil, fmt.Errorf("unknown cast type %s", t)
		}
	}

	s.stream = &Stream{Casts: s.casts}
	return s, err
}

// Ref implements v0.Component
func (s *Component) Ref() v0.ComponentRef {
	return s.ref
}

// Type implements v0.Component
func (s *Component) Type() v0.ComponentType {
	return Type
}

// Endpoints implements v0.Stateful
func (s *Component) Endpoints() (endpoints []mqhub.Endpoint) {
	endpoints = []mqhub.Endpoint{s.onOff, s.castTo, s.stateDp, s.recvDp}
	if s.imageDp != nil {
		endpoints = append(endpoints, s.imageDp)
	}
	return
}

// Start implements v0.LifecycleCtl
func (s *Component) Start() error {
	for _, c := range s.casts {
		if udpCast, ok := c.(*cmn.UDPCast); ok {
			if err := udpCast.Dial(); err != nil {
				return fmt.Errorf("start UDP cast error: %v", err)
			}
		}
	}
	s.stream.Start()
	s.stateDp.Update(&State{})
	s.recvDp.Update("")
	return nil
}

// Stop implements v0.LifecycleCtl
func (s *Component) Stop() error {
	s.stream.Stop()
	for _, c := range s.casts {
		if closer, ok := c.(io.Closer); ok {
			closer.Close()
		}
	}
	return nil
}

func (s *Component) setOn(on bool) {
	if on {
		opts, err := s.stream.On(s.settings)
		if err != nil {
			log.Printf("[%s] Turn on camera err: %v", s.ref.ComponentID(), err)
			return
		}
		s.stateDp.Update(&State{
			On:     true,
			Width:  opts.Width,
			Height: opts.Height,
			FourCC: opts.FourCC,
		})
	} else {
		if err := s.stream.Off(); err != nil {
			log.Printf("[%s] Turn off camera err: %v", s.ref.ComponentID(), err)
			return
		}
		s.stateDp.Update(&State{})
	}
}

func (s *Component) setCastTo(addr string) {
	if err := s.udpCast.SetRemoteAddr(addr); err != nil {
		log.Printf("[%s] SetRemoteAddr(%s) err: %v", s.ref.ComponentID(), addr, err)
		return
	}
	s.recvDp.Update(addr)
}

// Type is the Component type
var Type = eng.DefineComponentType("v4l2.camera",
	eng.ComponentFactoryFunc(func(ref v0.ComponentRef) (v0.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[V4L2] Camera").
	Register()

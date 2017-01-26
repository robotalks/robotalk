package camera

import (
	"fmt"

	"github.com/blackjack/webcam"
	"github.com/robotalks/mqhub.go/mqhub"
	eng "github.com/robotalks/robotalk/engine"
)

// Config defines camera configuration
type Config struct {
	Device string `json:"device"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Format string `json:"format"`
}

// State defines camera state
type State struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	FourCC FourCC `json:"fourcc"`
}

// Instance is the implementation
type Instance struct {
	config  Config
	state   State
	stateDp *mqhub.DataPoint
	imageDp *mqhub.DataPoint
	cam     *webcam.Webcam
}

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{
		config: Config{
			Device: "/dev/video0",
			Width:  640,
			Height: 480,
			Format: FourCCMJPG.String(),
		},
		stateDp: &mqhub.DataPoint{Name: "state", Retain: true},
		imageDp: &mqhub.DataPoint{Name: "image"},
	}
	err := spec.ConfigAs(&s.config)
	if err != nil {
		return nil, err
	}
	s.state.FourCC, err = ParseFourCC(s.config.Format)
	if err != nil {
		return nil, err
	}
	s.state.Width = s.config.Width
	s.state.Height = s.config.Height
	return s, nil
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Endpoints implements Stateful
func (s *Instance) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.stateDp, s.imageDp}
}

// Start implements LifecycleCtl
func (s *Instance) Start() error {
	cam, err := webcam.Open(s.config.Device)
	if err != nil {
		return err
	}

	f, w, h, err := cam.SetImageFormat(
		webcam.PixelFormat(s.state.FourCC),
		uint32(s.state.Width),
		uint32(s.state.Height))
	if err != nil {
		return err
	}
	if cc := FourCC(f); cc != s.state.FourCC {
		cam.Close()
		return fmt.Errorf("video format %s not supported, got %s",
			s.state.FourCC.String(), cc.String())
	}

	if err = cam.StartStreaming(); err != nil {
		cam.Close()
		return err
	}

	s.cam = cam
	s.state.Width = int(w)
	s.state.Height = int(h)

	go s.streaming(cam)
	return nil
}

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	s.cam.Close()
	return nil
}

func (s *Instance) streaming(cam *webcam.Webcam) {
	s.stateDp.Update(&s.state)
	for {
		err := cam.WaitForFrame(1)
		if err != nil {
			if _, ok := err.(*webcam.Timeout); !ok {
				break
			}
			continue
		}

		frame, _ := cam.ReadFrame()
		if frame == nil {
			continue
		}
		s.imageDp.Update(mqhub.StreamMessage(frame))
	}
}

// Type is the instance type
var Type = eng.DefineInstanceTypeAndRegister("v4l2.camera",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	}))

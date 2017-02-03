package camera

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"

	"github.com/blackjack/webcam"
	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/robotalk/common"
	eng "github.com/robotalks/robotalk/engine"
)

// Config defines camera configuration
type Config struct {
	Device  string            `json:"device"`
	Width   int               `json:"width"`
	Height  int               `json:"height"`
	Format  string            `json:"format"`
	Quality *int              `json:"quality"`
	Casts   map[string]string `json:"cast"`
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
	casts   []cmn.CastTarget
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
	if len(s.casts) == 0 {
		s.imageDp = &mqhub.DataPoint{Name: "image"}
		s.casts = append(s.casts, &cmn.DataPointCast{DP: s.imageDp})
	}
	return s, err
}

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Endpoints implements Stateful
func (s *Instance) Endpoints() (endpoints []mqhub.Endpoint) {
	endpoints = []mqhub.Endpoint{s.stateDp}
	if s.imageDp != nil {
		endpoints = append(endpoints, s.imageDp)
	}
	return
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

	for _, c := range s.casts {
		if udpCast, ok := c.(*cmn.UDPCast); ok {
			err = udpCast.Dial()
			if err != nil {
				cam.Close()
				return fmt.Errorf("start UDP cast error: %v", err)
			}
		}
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
	for _, c := range s.casts {
		if closer, ok := c.(io.Closer); ok {
			closer.Close()
		}
	}
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

		if s.state.FourCC == FourCCYUYV {
			// need jpeg encoding
			m := image.NewYCbCr(image.Rect(0, 0, s.state.Width, s.state.Height),
				image.YCbCrSubsampleRatio422)
			for i := 0; i < len(frame); i += 2 {
				n := i >> 1
				m.Y[n] = frame[i]
				if (n & 1) == 0 {
					m.Cb[n>>1] = frame[i+1]
				} else {
					m.Cr[n>>1] = frame[i+1]
				}
			}
			var jpg bytes.Buffer
			var opt *jpeg.Options
			if q := s.config.Quality; q != nil {
				opt = &jpeg.Options{Quality: *s.config.Quality}
			}
			if err = jpeg.Encode(&jpg, m, opt); err != nil {
				// TODO
			} else {
				frame = jpg.Bytes()
			}
		}

		for _, c := range s.casts {
			c.Cast(frame)
		}
	}
}

// Type is the instance type
var Type = eng.DefineInstanceType("v4l2.camera",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	})).
	Describe("[V4L2] Camera").
	Register()

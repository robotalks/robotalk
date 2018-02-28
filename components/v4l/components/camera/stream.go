package camera

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"

	"github.com/blackjack/webcam"
	cmn "github.com/robotalks/talk/common"
)

// Options is camera options
type Options struct {
	Device  string
	Width   int
	Height  int
	FourCC  FourCC
	Quality *int
}

// Camera is camera device
type Camera struct {
	Options
	cam *webcam.Webcam
}

// Open opens the camera device
func (s *Camera) Open() error {
	cam, err := webcam.Open(s.Device)
	if err != nil {
		return err
	}

	f, w, h, err := cam.SetImageFormat(
		webcam.PixelFormat(s.FourCC),
		uint32(s.Width),
		uint32(s.Height))
	if err != nil {
		return err
	}
	if cc := FourCC(f); cc != s.FourCC {
		cam.Close()
		return fmt.Errorf("video format %s not supported, got %s",
			s.FourCC.String(), cc.String())
	}

	if err = cam.StartStreaming(); err != nil {
		cam.Close()
		return err
	}

	s.cam = cam
	s.Width = int(w)
	s.Height = int(h)

	return nil
}

// Close closes the camera
func (s *Camera) Close() error {
	return s.cam.Close()
}

// GetFrame reads one frame
func (s *Camera) GetFrame() ([]byte, error) {
	err := s.cam.WaitForFrame(1)
	if err != nil {
		if _, ok := err.(*webcam.Timeout); ok {
			return nil, nil
		}
		return nil, err
	}

	frame, _ := s.cam.ReadFrame()
	if frame == nil {
		return nil, nil
	}

	if s.FourCC == FourCCYUYV {
		// need jpeg encoding
		m := image.NewYCbCr(image.Rect(0, 0, s.Width, s.Height),
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
		if q := s.Quality; q != nil {
			opt = &jpeg.Options{Quality: *s.Quality}
		}
		if err = jpeg.Encode(&jpg, m, opt); err != nil {
			return nil, err
		} else {
			frame = jpg.Bytes()
		}
	}

	return frame, nil
}

// Stream is camera streamer
type Stream struct {
	Casts []cmn.CastTarget

	cam     *Camera
	frameCh chan []byte
	opCh    chan func()
	stopCh  chan struct{}
}

// Start starts the background streamer
func (s *Stream) Start() {
	s.frameCh = make(chan []byte, 1)
	s.opCh = make(chan func())
	s.stopCh = make(chan struct{})
	go s.run(s.frameCh, s.opCh, s.stopCh)
}

// Stop stops the background streamer
func (s *Stream) Stop() {
	s.Off()
	if ch := s.opCh; ch != nil {
		s.opCh = nil
		close(ch)
		<-s.stopCh
	}
}

// On turns on camera
func (s *Stream) On(settings Options) (opts Options, err error) {
	err = s.Do(func() error {
		if cam := s.cam; cam == nil {
			cam = &Camera{Options: settings}
			if err = cam.Open(); err != nil {
				return err
			}
			s.cam = cam
			opts = cam.Options
			go s.stream(cam)
		}
		return nil
	})
	return
}

// Off turns off camera
func (s *Stream) Off() error {
	return s.Do(func() error {
		if cam := s.cam; cam != nil {
			s.cam = nil
			cam.Close()
		}
		return nil
	})
}

// Do runs an operation in stream task
func (s *Stream) Do(fn func() error) error {
	if ch := s.opCh; ch != nil {
		errCh := make(chan error)
		ch <- func() {
			errCh <- fn()
		}
		return <-errCh
	}
	return nil
}

func (s *Stream) run(frameCh <-chan []byte, opCh <-chan func(), stopCh chan struct{}) {
	defer func() {
		if cam := s.cam; cam != nil {
			s.cam = nil
			cam.Close()
		}
		close(stopCh)
	}()
	for {
		select {
		case frame := <-frameCh:
			for _, c := range s.Casts {
				c.Cast(frame)
			}
		case op, ok := <-opCh:
			if !ok {
				return
			}
			op()
		}
	}
}

func (s *Stream) stream(cam *Camera) {
	for {
		frame, err := cam.GetFrame()
		if err != nil {
			log.Printf("Camera Stream STOP: %v", err)
			cam.Close()
			break
		}
		if frame != nil {
			s.frameCh <- frame
		}
	}
}

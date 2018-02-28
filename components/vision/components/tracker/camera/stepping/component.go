package stepping

import (
	"fmt"
	"math"
	"strings"

	"github.com/robotalks/mqhub.go/mqhub"
	"github.com/robotalks/talk-vision/utils"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
)

// Dir is enumeration of servo direction
type Dir int

// Directions
const (
	// Pan moves servo horizontally
	Pan Dir = iota
	// Tilt moves servo vertically
	Tilt
)

// Axis
const (
	AxisX string = "x"
	AxisY string = "y"
)

// Component is the implementation
type Component struct {
	Axis      string            `map:"axis"`
	AngleMin  *float32          `map:"angle-min"`
	AngleMax  *float32          `map:"angle-max"`
	AngleStep *float32          `map:"angle-step"`
	Tolerance *float32          `map:"tolerance"`
	Objects   mqhub.EndpointRef `inject:"objects" map:"-"`
	Servo     mqhub.EndpointRef `inject:"servo" map:"-"`

	ref talk.ComponentRef

	dir                 Dir
	mirror              bool
	cur, min, max, step float32
	tolerance           float32
	watcher             mqhub.Watcher
}

func mapAngle(a float32) (float32, error) {
	if a < -90 || a > 90 {
		return 0, fmt.Errorf("angle %v out of range", a)
	}
	return (a+90)/90 - 1, nil
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{ref: ref}
	err := eng.SetupComponent(s, ref)
	if err != nil {
		return nil, err
	}

	axis := strings.ToLower(s.Axis)
	if strings.HasPrefix("-", axis) {
		s.mirror = true
		axis = axis[1:]
	}
	switch axis {
	case AxisX:
		s.dir = Pan
	case AxisY:
		s.dir = Tilt
	default:
		return nil, fmt.Errorf("invalid axis %s", s.Axis)
	}

	if s.AngleMin != nil {
		if s.min, err = mapAngle(*s.AngleMin); err != nil {
			return nil, err
		}
	} else {
		s.min = -90
	}
	if s.AngleMax != nil {
		if s.max, err = mapAngle(*s.AngleMax); err != nil {
			return nil, err
		}
	} else {
		s.max = 90
	}
	if s.AngleStep != nil {
		s.step = *s.AngleStep / 90
	} else {
		s.step = 1.0 / 90
	}

	if s.Tolerance != nil {
		s.tolerance = *s.Tolerance
		if s.tolerance < 0 || s.tolerance > 1 {
			return nil, fmt.Errorf("invalid tolerance %v (should be 0 - 1)", s.tolerance)
		}
	} else {
		s.tolerance = 0.5
	}

	return s, nil
}

// Ref implements talk.Component
func (s *Component) Ref() talk.ComponentRef {
	return s.ref
}

// Type implements talk.Component
func (s *Component) Type() talk.ComponentType {
	return Type
}

// Start implements talk.LifecycleCtl
func (s *Component) Start() (err error) {
	if s.watcher, err = s.Objects.Watch(s.messageSink()); err != nil {
		return
	}
	s.setPos()
	return
}

// Stop implements talk.LifecycleCtl
func (s *Component) Stop() error {
	s.watcher.Close()
	return nil
}

func (s *Component) setPos() {
	s.Servo.ConsumeMessage(mqhub.MsgFrom(s.cur))
}

type rangeFn func(*utils.Size, *utils.Rect) (float32, float32)

func (s *Component) messageSink() mqhub.MessageSink {
	var f rangeFn
	if s.dir == Pan {
		f = func(s *utils.Size, r *utils.Rect) (float32, float32) {
			return float32(r.X*2)/float32(s.W) - 1, float32((r.X+r.W)*2)/float32(s.W) - 1
		}
	} else {
		f = func(s *utils.Size, r *utils.Rect) (float32, float32) {
			return 1 - float32(r.Y*2)/float32(s.H), 1 - float32((r.Y+r.H)*2)/float32(s.H)
		}
	}
	return mqhub.MessageSinkAs(func(r *utils.Result) {
		if len(r.Objects) == 0 || r.Objects[0] == nil {
			return
		}
		a0, a1 := f(&r.Size, &r.Objects[0].Range)
		if s.mirror {
			a0, a1 = -a0, -a1
		}
		cntr := (a0 + a1) / 2
		t := float32(math.Abs(float64((a1 - a0) * s.tolerance / 2)))
		cur := s.cur
		switch {
		case cntr < -t:
			if cur += s.step; cur > 1 {
				cur = 1
			}
		case cntr > t:
			if cur -= s.step; cur < -1 {
				cur = -1
			}
		}
		if cur != s.cur {
			s.cur = cur
			s.setPos()
		}
	})
}

// Type is the component type
var Type = eng.DefineComponentType("vision.tracker.camera.stepping",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[Vision] Track Object using Simple Tiny Stepping").
	Register()

package stepping

import (
	"fmt"
	"math"
	"strings"

	"github.com/robotalks/mqhub.go/mqhub"
	eng "github.com/robotalks/robotalk/engine"
	"github.com/robotalks/robotalk/logics/vision/utils"
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

// Instance implements camera tracker
type Instance struct {
	Axis      string            `json:"axis"`
	AngleMin  *float32          `json:"angle-min"`
	AngleMax  *float32          `json:"angle-max"`
	AngleStep *float32          `json:"angle-step"`
	Tolerance *float32          `json:"tolerance"`
	Objects   mqhub.EndpointRef `key:"objects" json:"-"`
	Servo     mqhub.EndpointRef `key:"servo" json:"-"`

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

// NewInstance creates an instance
func NewInstance(spec *eng.ComponentSpec) (*Instance, error) {
	s := &Instance{}
	err := spec.Reflect(s)
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

// Type implements Instance
func (s *Instance) Type() eng.InstanceType {
	return Type
}

// Start implements LifecycleCtl
func (s *Instance) Start() (err error) {
	if s.watcher, err = s.Objects.Watch(s.messageSink()); err != nil {
		return
	}
	s.setPos()
	return
}

// Stop implements LifecycleCtl
func (s *Instance) Stop() error {
	s.watcher.Close()
	return nil
}

func (s *Instance) setPos() {
	s.Servo.ConsumeMessage(mqhub.MsgFrom(s.cur))
}

type rangeFn func(*utils.Size, *utils.Rect) (float32, float32)

func (s *Instance) messageSink() mqhub.MessageSink {
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

// Type is the instance type
var Type = eng.DefineInstanceTypeAndRegister("vision.tracker.camera.stepping",
	eng.InstanceFactoryFunc(func(spec *eng.ComponentSpec) (eng.Instance, error) {
		return NewInstance(spec)
	}))

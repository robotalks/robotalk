package buzzer

import (
	"fmt"
	"runtime"
	"time"

	"github.com/robotalks/mqhub.go/mqhub"
	cmn "github.com/robotalks/talk-gobot/common"
	talk "github.com/robotalks/talk.contract/v0"
	eng "github.com/robotalks/talk/engine"
	"gobot.io/x/gobot/drivers/gpio"
)

// Config defines buzz configuration
type Config struct {
	Pin string `map:"pin"`
}

// Component is the implement of buzz Component
type Component struct {
	Config
	Adapter cmn.Adapter `inject:"gpio" map:"-"`

	ref      talk.ComponentRef
	device   *gpio.BuzzerDriver
	playing  *mqhub.DataPoint
	writeSeq *mqhub.Reactor
	seqCh    chan []float32
	stopCh   chan struct{}
}

// NewComponent creates a Component
func NewComponent(ref talk.ComponentRef) (talk.Component, error) {
	s := &Component{ref: ref, playing: &mqhub.DataPoint{Name: "playing", Retain: true}}
	s.writeSeq = mqhub.ReactorAs("seq", s.playSeq)
	if err := eng.SetupComponent(s, ref); err != nil {
		return nil, err
	}
	conn, ok := s.Adapter.Adaptor().(gpio.DigitalWriter)
	if !ok {
		return nil, fmt.Errorf("injection adapter of %s is not gpio.DigitalWriter", ref.MessagePath())
	}
	s.device = gpio.NewBuzzerDriver(conn, s.Pin)
	if err := s.device.Start(); err != nil {
		return nil, err
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

// Endpoints implements talk.Stateful
func (s *Component) Endpoints() []mqhub.Endpoint {
	return []mqhub.Endpoint{s.writeSeq, s.playing}
}

// Start implements talk.LifecycleCtl
func (s *Component) Start() error {
	s.seqCh = make(chan []float32, 1)
	s.stopCh = make(chan struct{})
	go s.playTask(s.seqCh, s.stopCh)
	return nil
}

// Stop implements talk.LifecycleCtl
func (s *Component) Stop() error {
	if ch := s.seqCh; ch != nil {
		s.seqCh = nil
		close(ch)
		<-s.stopCh
	}
	return nil
}

func (s *Component) playSeq(v []float32) {
	if ch := s.seqCh; ch != nil {
		ch <- v
	}
}

func (s *Component) playTask(seqCh <-chan []float32, stopCh chan struct{}) {
	var seq []float32
	var ticks, maxTicks uint64
	cur, tone := 0, time.Hour
	var playing bool
	s.playing.Update(playing)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for {
		select {
		case newSeq, ok := <-seqCh:
			if !ok {
				s.device.Off()
				s.playing.Update(false)
				close(stopCh)
				return
			}
			seq, cur, tone = newSeq, 0, time.Duration(0)
			ticks, maxTicks = 0, 0
		case <-time.After(tone):
			if ticks < maxTicks {
				if (ticks & 1) == 0 {
					s.device.On()
				} else {
					s.device.Off()
				}
				ticks++
				if ticks < maxTicks {
					continue
				}
			}
			s.device.Off()
			if cur < len(seq)-1 {
				hz := seq[cur]
				if hz <= 0 {
					tone = time.Duration(seq[cur+1]) * time.Millisecond
					ticks, maxTicks = 0, 0
				} else {
					ms := (1.0 / (2.0 * hz)) * 1000.0
					maxTicks = uint64(seq[cur+1] / ms)
					if (maxTicks & 1) != 0 {
						maxTicks++
					}
					ticks = 0
					tone = time.Duration(ms*1000.0) * time.Microsecond
				}
				cur += 2
			} else {
				tone = time.Hour
			}

			if nowPlaying := ticks < maxTicks; nowPlaying != playing {
				playing = nowPlaying
				s.playing.Update(playing)
			}
		}
	}
}

// Type is the Component type
var Type = eng.DefineComponentType("gobot.gpio.buzzer",
	eng.ComponentFactoryFunc(func(ref talk.ComponentRef) (talk.Component, error) {
		return NewComponent(ref)
	})).
	Describe("[GoBot] GPIO Buzzer (Digital Pin)").
	Register()

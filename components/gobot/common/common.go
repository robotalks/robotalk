package common

import "gobot.io/x/gobot"

// Adapter represents gobot Adaptor
type Adapter interface {
	Adaptor() gobot.Adaptor
}

// Servo defines abstract servo component
type Servo interface {
	// SetServoPos sets the position -1.0 - 1.0
	SetServoPos(pos float32) error
}

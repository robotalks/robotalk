package common

import "gobot.io/x/gobot"

// Adapter represents gobot Adaptor
type Adapter interface {
	Adaptor() gobot.Adaptor
}

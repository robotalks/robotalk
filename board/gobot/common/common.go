package common

import (
	"github.com/easeway/langx.go/errors"
	"gobot.io/x/gobot"
)

// Errs converts error slice into aggregated error
func Errs(errs []error) error {
	aggregated := &errors.AggregatedError{}
	return aggregated.AddMany(errs...).Aggregate()
}

// Adapter represents gobot Adaptor
type Adapter interface {
	Adaptor() gobot.Adaptor
}

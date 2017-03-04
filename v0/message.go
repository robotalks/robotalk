package talk

import "io"

// Message wraps abstract messages
type Message interface {
	// Path is the path of destination
	Path() string
	// Bytes retrieves raw message data
	Bytes() []byte
	// Value retrieves the actual wrapped value if present
	Value() (interface{}, bool)
}

// ErrorChan is receiving chan for error
type ErrorChan <-chan error

// MessageDispatcher dispatches message
type MessageDispatcher interface {
	DispatchMessage(Message) ErrorChan
}

// MessageSource emits messages
type MessageSource interface {
	WatchMessage(MessageDispatcher) (MessageWatcher, error)
}

// MessageWatcher represents a subscription of messages
type MessageWatcher interface {
	io.Closer
	MessageSource() MessageSource
}

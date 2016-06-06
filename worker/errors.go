package worker

import (
	"errors"
)

var (
	errNoDispatch    = errors.New("no dispatch function found for event")
	errMaxQueueDepth = errors.New("queue is full")
)

package producer

import (
	"github.com/opsee/gmunch"
)

type Producer interface {
	Publish(*gmunch.Event) error
}

type Config interface {
	isProducerConfig()
}

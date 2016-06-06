package debug

import (
	"github.com/opsee/gmunch"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type Job struct {
	event   *gmunch.Event
	context context.Context
}

func New(evt *gmunch.Event) *Job {
	return &Job{
		event:   evt,
		context: context.Background(),
	}
}

func (j *Job) Context() context.Context {
	return j.context
}

func (j *Job) Execute() (interface{}, error) {
	log.Infof("job: %s", j.event.Name)

	stuff := make(map[string]interface{})
	err := j.event.Decoder().Decode(&stuff)
	if err != nil {
		log.WithError(err).Error("couldn't decode fields")
		return nil, err
	}

	log.Infof("fields: %#v", stuff)
	return struct{}{}, nil
}

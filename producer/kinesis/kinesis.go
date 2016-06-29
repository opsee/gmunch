package kinesis

import (
	"fmt"
	"math/rand"
	"time"

	log "github.com/opsee/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/golang/protobuf/proto"
	"github.com/opsee/gmunch"
)

type producer struct {
	stream string
	rand   *rand.Rand
	client *kinesis.Kinesis
}

type Config struct {
	Stream string
}

func New(config Config) *producer {
	return &producer{
		stream: config.Stream,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		client: kinesis.New(session.New()),
	}
}

func (p *producer) Publish(event *gmunch.Event) error {
	pbdata, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	resp, err := p.client.PutRecord(&kinesis.PutRecordInput{
		StreamName:   aws.String(p.stream),
		Data:         pbdata,
		PartitionKey: aws.String(fmt.Sprintf("%s-%d", event.Name, p.rand.Int63())),
	})

	log.WithField("producer", "kinesis").Debugf("put record response: %#v", resp)

	return err
}

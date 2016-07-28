package kinesis

import (
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/cenkalti/backoff"
	etcd "github.com/coreos/etcd/client"
	"github.com/golang/protobuf/proto"
	"github.com/opsee/gmunch"
	log "github.com/opsee/logrus"
	"golang.org/x/net/context"
)

const (
	flushIntervalDuration = 10 * time.Second
	sleepDuration         = 500 * time.Millisecond
)

type kinesisConsumer struct {
	stream        string
	shardId       *string
	shardPath     string
	etcdEndpoints []string
	etcd          etcd.KeysAPI
	iterator      *string
	iteratorMut   sync.Mutex
	sequence      *string
	client        *kinesis.Kinesis
	stopChan      chan struct{}
	stoppedChan   chan struct{}
	stopping      bool
	eventChan     chan *gmunch.Event
	logger        *log.Entry
}

type Config struct {
	Stream        string
	EtcdEndpoints []string
	ShardPath     string
	Region        string
}

func New(config Config) *kinesisConsumer {
	return &kinesisConsumer{
		stream:        config.Stream,
		etcdEndpoints: config.EtcdEndpoints,
		client:        kinesis.New(session.New(aws.NewConfig().WithRegion(config.Region))),
		stopChan:      make(chan struct{}, 1),
		stoppedChan:   make(chan struct{}, 1),
		eventChan:     make(chan *gmunch.Event),
		shardPath:     config.ShardPath,
		logger:        log.WithField("consumer", "kinesis"),
	}
}

func (c *kinesisConsumer) Start() error {
	var err error
	c.logger.Info("starting")

	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               c.etcdEndpoints,
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	})

	if err != nil {
		return err
	}

	c.etcd = etcd.NewKeysAPI(etcdClient)

	out, err := c.client.DescribeStream(&kinesis.DescribeStreamInput{
		StreamName: aws.String(c.stream),
	})

	if err != nil {
		return err
	}

	if out.StreamDescription == nil {
		return fmt.Errorf("no stream found in kinesis")
	}

	if len(out.StreamDescription.Shards) == 0 {
		return fmt.Errorf("no shards found in kinesis stream")
	}

	// we're just hardcoding a single shard ok
	c.shardId = out.StreamDescription.Shards[0].ShardId
	if c.shardId == nil {
		return fmt.Errorf("no shard id found")
	}

	err = c.getIterator()
	if err != nil {
		return err
	}

	go c.flushIterator()
	c.run()
	return nil
}

func (c *kinesisConsumer) Stop() {
	c.logger.Info("stopping")
	defer close(c.stopChan)
	defer close(c.stoppedChan)

	c.stopChan <- struct{}{}
	select {
	case <-c.stoppedChan:
	case <-time.After(5 * time.Second):
	}
	c.logger.Info("stopped")
}

func (c *kinesisConsumer) Events() chan *gmunch.Event {
	return c.eventChan
}

func (c *kinesisConsumer) shouldStop() bool {
	if c.stopping {
		return true
	}

	select {
	case <-c.stopChan:
		c.stopping = true
		return c.stopping
	default:
		return false
	}
}

func (c *kinesisConsumer) run() {
	for {
		var (
			err error
			out *kinesis.GetRecordsOutput
		)

		backoff.Retry(func() error {
			if c.shouldStop() {
				return nil
			}

			c.logger.Debugf("getting records with iterator: %s", aws.StringValue(c.iterator))

			out, err = c.client.GetRecords(&kinesis.GetRecordsInput{
				ShardIterator: c.iterator,
				Limit:         aws.Int64(1),
			})

			if err != nil {
				c.logger.WithError(err).Error("AWS error")
				return err
			}

			return nil

		}, &backoff.ExponentialBackOff{
			InitialInterval:     100 * time.Millisecond,
			RandomizationFactor: 0.5,
			Multiplier:          1.5,
			MaxInterval:         1 * time.Minute,
			MaxElapsedTime:      60 * time.Minute,
			Clock:               &systemClock{},
		})

		// check for shutdown signal
		if c.shouldStop() {
			goto SHUTDOWN
		}

		// we couldn't recover after 60 mins
		if err != nil {
			goto SHUTDOWN
		}

		for _, rec := range out.Records {
			event := &gmunch.Event{}
			err = proto.Unmarshal(rec.Data, event)

			// ignore unmarshaling errors, if you can't send the right kind of data, then
			// continue incrementing the sequence and to heck with you
			if err != nil {
				c.logger.WithError(err).Error("proto unmarshal error")
				continue
			}

			if c.shouldStop() {
				goto SHUTDOWN
			}

			c.logger.WithField("name", event.Name).Debug("sending event to event channel")
			c.eventChan <- event
			c.sequence = rec.SequenceNumber
		}

		// our shard has been closed
		if out.NextShardIterator == nil {
			c.logger.Info("shard has been closed")
			goto SHUTDOWN
		}

		c.logger.Debugf("setting next iterator: %s", aws.StringValue(out.NextShardIterator))
		c.setIterator(out.NextShardIterator)

		// if there aren't any more records, just chill for a bit. ideally this would be adaptive
		if aws.Int64Value(out.MillisBehindLatest) == 0 {
			time.Sleep(sleepDuration)
		}
	}

SHUTDOWN:
	close(c.eventChan)
	// persist cursor
	c.putSequence()
	c.stoppedChan <- struct{}{}
}

func (c *kinesisConsumer) flushIterator() {
	for {
		select {
		case <-time.After(flushIntervalDuration):
			c.putSequence()
		}
	}
}

func (c *kinesisConsumer) setIterator(iter *string) {
	c.iteratorMut.Lock()
	defer c.iteratorMut.Unlock()
	c.iterator = iter
}

func (c *kinesisConsumer) putSequence() error {
	_, err := c.etcd.Set(context.Background(), path.Join(c.shardPath, aws.StringValue(c.shardId), "sequence"), aws.StringValue(c.sequence), &etcd.SetOptions{})
	return err
}

func (c *kinesisConsumer) getIterator() error {
	response, err := c.etcd.Get(context.Background(), path.Join(c.shardPath, aws.StringValue(c.shardId), "sequence"), &etcd.GetOptions{
		Quorum: true,
	})

	if err != nil {
		if etcdErr, ok := err.(etcd.Error); ok {
			switch etcdErr.Code {
			case etcd.ErrorCodeKeyNotFound:
				return c.getIteratorHorizon()
			}
		}

		c.logger.WithError(err).Error("etcd error")
		return err
	}

	out, err := c.client.GetShardIterator(&kinesis.GetShardIteratorInput{
		ShardId:                c.shardId,
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StreamName:             aws.String(c.stream),
		StartingSequenceNumber: aws.String(response.Node.Value),
	})

	if err != nil {
		c.logger.WithError(err).Error("AWS error")
		return err
	}

	c.setIterator(out.ShardIterator)
	return nil
}

func (c *kinesisConsumer) getIteratorHorizon() error {
	out, err := c.client.GetShardIterator(&kinesis.GetShardIteratorInput{
		ShardId:           c.shardId,
		ShardIteratorType: aws.String(kinesis.ShardIteratorTypeTrimHorizon),
		StreamName:        aws.String(c.stream),
	})

	if err != nil {
		c.logger.WithError(err).Error("AWS error")
		return err
	}

	c.setIterator(out.ShardIterator)
	return nil
}

type systemClock struct{}

func (s *systemClock) Now() time.Time {
	return time.Now()
}

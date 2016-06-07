package nsq

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/protobuf/proto"
	"github.com/nsqio/go-nsq"
	"github.com/opsee/gmunch"
)

type nsqConsumer struct {
	config      *Config
	consumer    *nsq.Consumer
	eventChan   chan *gmunch.Event
	stopChan    chan struct{}
	stoppedChan chan struct{}
	logger      *log.Entry
}

type Config struct {
	Topic            string
	Channel          string
	LookupdAddresses []string
	NSQConfig        *nsq.Config
	HandlerCount     int
}

func New(config Config) *nsqConsumer {
	return &nsqConsumer{
		config:      &config,
		stopChan:    make(chan struct{}, 1),
		stoppedChan: make(chan struct{}, 1),
		eventChan:   make(chan *gmunch.Event),
		logger:      log.WithField("consumer", "nsq"),
	}
}

func (c *nsqConsumer) Start() error {
	var err error

	if c.config.NSQConfig == nil {
		c.logger.Info("no nsq config detected, setting max_in_flight to 4")
		c.config.NSQConfig = nsq.NewConfig()
		c.config.NSQConfig.MaxInFlight = 4
	}

	c.consumer, err = nsq.NewConsumer(c.config.Topic, c.config.Channel, c.config.NSQConfig)
	if err != nil {
		log.WithError(err).Error("couldn't create nsq consumer")
		return err
	}

	if c.config.HandlerCount == 0 {
		c.logger.Info("no nsq handler count config detected, setting to 4")
		c.config.HandlerCount = 4
	}

	c.consumer.AddConcurrentHandlers(c, c.config.HandlerCount)
	c.consumer.ConnectToNSQLookupds(c.config.LookupdAddresses)

	<-c.stopChan
	c.consumer.Stop()
	<-c.consumer.StopChan
	c.stoppedChan <- struct{}{}

	return nil
}

func (c *nsqConsumer) Stop() {
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

func (c *nsqConsumer) Events() chan *gmunch.Event {
	return c.eventChan
}

func (c *nsqConsumer) HandleMessage(m *nsq.Message) error {
	event := &gmunch.Event{}
	err := proto.Unmarshal(m.Body, event)
	if err != nil {
		c.logger.WithError(err).Error("couldn't unmarshal gmunch event")
		return err
	}

	c.eventChan <- event

	return nil
}

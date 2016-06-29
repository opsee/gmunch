package server

import (
	"net"

	log "github.com/opsee/logrus"
	"github.com/opsee/gmunch"
	"github.com/opsee/gmunch/producer"
	"github.com/opsee/gmunch/worker"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	grpcauth "google.golang.org/grpc/credentials"
)

type server struct {
	server   *grpc.Server
	producer producer.Producer
	worker   *worker.Worker
}

type Config struct {
	LogLevel string
	Producer producer.Producer
	Consumer worker.Consumer
	Dispatch worker.Dispatch
	MaxJobs  uint
}

func New(config Config) *server {
	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Warnf("couldn't parse log level: %s", config.LogLevel)
	} else {
		log.SetLevel(level)
	}

	return &server{
		producer: config.Producer,
		worker: worker.New(worker.Config{
			Consumer: config.Consumer,
			Dispatch: config.Dispatch,
			MaxJobs:  config.MaxJobs,
		}),
	}
}

func (s *server) Start(listenAddr, cert, certkey string) error {
	go s.worker.Start()

	auth, err := grpcauth.NewServerTLSFromFile(cert, certkey)
	if err != nil {
		return err
	}

	s.server = grpc.NewServer(grpc.Creds(auth))
	gmunch.RegisterEventsServer(s.server, s)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	return s.server.Serve(lis)
}

func (s *server) Publish(ctx context.Context, event *gmunch.Event) (*gmunch.Response, error) {
	if event == nil {
		return nil, errNoEvent
	}

	if err := s.producer.Publish(event); err != nil {
		return nil, err
	}

	return &gmunch.Response{Ok: true}, nil
}

func (s *server) Stop() {
	s.worker.Stop()
	s.server.Stop()
}

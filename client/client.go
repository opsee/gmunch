package client

import (
	"github.com/opsee/gmunch"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Client interface {
	Send(name string, data interface{}) error
}

type client struct {
	grpcClient gmunch.EventsClient
}

func New(addr string, creds credentials.TransportAuthenticator) (Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return &client{gmunch.NewEventsClient(conn)}, nil
}

func (c *client) Send(name string, data interface{}) error {
	event := &gmunch.Event{Name: name}

	err := event.EncodeData(data)
	if err != nil {
		return err
	}

	_, err = c.grpcClient.Publish(context.Background(), event)

	return err
}

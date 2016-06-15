package client

import (
	"crypto/tls"

	"github.com/opsee/gmunch"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Client interface {
	// Send() is used for enqueueing an event via gmunch. The name clearly identifies
	// the event type so that your worker process can subscribe handlers for that event.
	// Names do not have to be globally unique--simply unique per gmunch instance (one or
	// more gmunch Servers that use the same configuration).
	Send(name string, data interface{}) error
}

// ClientConfig objects are used to configure the transport's client.
type Config struct {
	// TLSConfig must be provided.
	TLSConfig tls.Config
}

type client struct {
	grpcClient gmunch.EventsClient
}

func New(addr string, config Config) (Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(credentials.NewTLS(&config.TLSConfig)))
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

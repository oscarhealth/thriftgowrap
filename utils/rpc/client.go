package rpc

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/oscarhealth/thriftgowrap/utils/retry"
)
// TransportFactory is an interface for returning a thrift client with opened transport.
type TransportFactory interface {
	GetTransport() (thrift.TTransport, thrift.TProtocolFactory, error)
}

// Client is used to implement RPC calls. This should be type-aliased for specific clients.
type Client struct {
	TransportFactory TransportFactory
	Retrier          *retry.Retrier
}

// NewClient creates a new Client.
func NewClient(transportFactory TransportFactory, options ...ClientOption) *Client {
	client := &Client{
		TransportFactory: transportFactory,
		Retrier:          retry.NewRetrier(),
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// ClientOption is a function that configures RPCClient
type ClientOption func(c *Client)

// RetrierOption sets RPCClient.Retrier
// Defaults to no-op Retrier
func RetrierOption(retrier *retry.Retrier) ClientOption {
	return func(client *Client) {
		client.Retrier = retrier
	}
}

package clusterbox

import (
	"context"
	"net"
)

// Service reachable via an endpoint
type Service interface {
	// Endpoint to reach the service at
	Endpoint() string
}

// Node that a ClusterBox can manage
type Node interface {
	Service

	// Setup sets the node up for work
	//
	// It probably needs to know some of the other node endpoints,
	// so endpoints list is passed with the full cluster node's endpoints
	Setup(endpoints []string)

	// Serve is a blocking call that runs the server loop.
	//
	// It runs until the Node is cancelled or the server is
	// done somehow; completes, fails or panics
	Serve()

	// Client is a blocking call that runs the client loop.
	//
	// It runs until the Node is cancelled or the client is
	// done somehow; completes, fails or panics
	Client()
}

// NewNodeFunc creates a fresh Node for ClusterBox
//
// Accepts as inputs a cancellable context & the node number in the cluster.
type NewNodeFunc func(context.Context, int) (Node, error)

// NetService Net service base for Nodes.
type NetService struct {
	ln net.Listener
}

// Endpoint returns the NetService's endpoint
func (hn *NetService) Endpoint() string {
	return hn.ln.Addr().String()
}

// NewNetService creates a NetService or returns an error if that fails
func NewNetService(network string) (*NetService, error) {
	ln, err := net.Listen(network, ":0")
	if err != nil {
		return nil, err
	}
	return &NetService{ln: ln}, nil
}

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

// TCPService TCP service base for Nodes.
type TCPService struct {
	ln   net.Listener
	cctx context.Context
}

// Endpoint returns this HTTPNode's endpoint
func (hn *TCPService) Endpoint() string {
	return hn.ln.Addr().String()
}

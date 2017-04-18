package clusterbox

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
)

// Node that a ClusterBox can manage
type Node interface {
	// Endpoint returns the node's endpoint.
	Endpoint() string

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

// HTTPNode partly implements Node interface for HTTP.
//
// Consummers embedding HTTPNode must provide implementations
// for Setup(), Serve() & Client() if they want to use it as a node.
type HTTPNode struct {
	ln   net.Listener
	cctx context.Context
}

// Endpoint returns this HTTPNode's endpoint
func (hn *HTTPNode) Endpoint() string {
	return hn.ln.Addr().String()
}

// ServeWith runs the HTTPNode server's side by routing to
// the given ServeMux handlers
func (hn *HTTPNode) ServeWith(serveMux *http.ServeMux) {
	endpoint := hn.Endpoint()
	err := http.Serve(hn.ln, serveMux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s serve() died with: %v\n", endpoint, err)
	}
}

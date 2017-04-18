package clusterbox

import (
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
	Setup(nodes []Node)
	// Serve is a blocking call that runs the server loop.
	//
	// It blocks until the service completes or the Node is Stopped
	Serve()

	// ClientLoop is a blocking call that runs the client loop.
	//
	// It blocks until completed or the Node is stopped
	Client()

	// Stop the Node
	Stop() error
}

// NewNodeFunc creates a fresh Node for ClusterBox
type NewNodeFunc func(int) (Node, error)

// HTTPNode partly implements Node interface for HTTP.
//
// Consummers embedding HTTPNode must provide implementations
// for Setup(), Serve() & Client() if they want to use it as a node.
type HTTPNode struct {
	ln         net.Listener
	clientDone chan struct{}
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

// Stop the HTTPNode work (Serve & ClientLoop)
func (hn *HTTPNode) Stop() error {
	select {
	case <-hn.clientDone:
		// when ctx is already closed
	default:
		// when ctx is not closed yet
		close(hn.clientDone)
	}
	return hn.ln.Close()
}

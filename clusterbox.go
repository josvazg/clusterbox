package clusterbox

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
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
	ClientLoop()

	// Stop the Node
	Stop() error
}

// NewNodeFunc creates a fresh Node for ClusterBox
type NewNodeFunc func(int) (Node, error)

// ClusterBox allows you to run a 'full' cluster in a box.
type ClusterBox struct {
	nodes []Node
	wg    sync.WaitGroup
}

// NewClusterBox creates a ClusterBox of the given size
func NewClusterBox(size int, newNode NewNodeFunc) (*ClusterBox, error) {
	nodes := make([]Node, 0, size)
	for i := 0; i < size; i++ {
		node, err := newNode(i)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
		fmt.Printf("Node %d listens at %s\n", i, node.Endpoint())
	}
	return &ClusterBox{nodes: nodes}, nil
}

// Run the ClusterBox until all nodes stop
func (c *ClusterBox) Run() {
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	go func() {
		<-interrupts
		fmt.Println("Received Ctrl+C, stopping nodes...")
		for _, n := range c.nodes {
			n.Stop()
		}
	}()
	for _, n := range c.nodes {
		c.wg.Add(1)
		go func(n Node) {
			n.Setup(c.nodes)
			serverDone := make(chan struct{})
			go func(n Node, serverDone chan struct{}) {
				n.Serve()
				fmt.Printf("%s server is done\n", n.Endpoint())
				close(serverDone)
			}(n, serverDone)
			clientDone := make(chan struct{})
			go func(n Node, clientDone chan struct{}) {
				n.ClientLoop()
				fmt.Printf("%s client is done\n", n.Endpoint())
				close(clientDone)
			}(n, clientDone)
			var peerDone chan struct{}
			select {
			case <-serverDone:
				peerDone = clientDone
				fmt.Printf("%s waits for client to close...\n", n.Endpoint())
			case <-clientDone:
				peerDone = serverDone
				fmt.Printf("%s waits for server to close...\n", n.Endpoint())
			}
			n.Stop()
			<-peerDone
			fmt.Printf("%s closed both client&server\n", n.Endpoint())
			c.wg.Done()
		}(n)
	}
	c.wg.Wait()
}
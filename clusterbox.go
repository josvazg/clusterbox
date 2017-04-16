package clusterbox

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

// Node that a ClusterBox can manage
type Node interface {
	Endpoint() string                                                               // return the node endpoint
	Setup(cancelContext context.Context, cancelFn context.CancelFunc, nodes []Node) // setup the node
	Serve()                                                                         // run the server loop, block until complete or stopped
	ClientLoop()                                                                    // run the client loop, until complete or stopped
	Stop() error                                                                    // stop the Node, should make Serve() and ClientLoop() end if they were still running
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
	ctx, cancel := context.WithCancel(context.Background())
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	go func() {
		<-interrupts
		fmt.Println("Received Ctrl+C, stopping nodes...")
		cancel()
	}()
	for _, n := range c.nodes {
		c.wg.Add(1)
		go func(ctx context.Context, n Node) {
			nodeCtx, cancelNode := context.WithCancel(ctx)
			n.Setup(nodeCtx, cancelNode, c.nodes)
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
			cancelNode()
		}(ctx, n)
	}
	c.wg.Wait()
	cancel()
}

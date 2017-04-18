package clusterbox

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

// ClusterBox allows you to run a 'full' cluster in a box.
type ClusterBox struct {
	ctx     context.Context
	nodes   []Node
	cancels []context.CancelFunc
	wg      sync.WaitGroup
}

// NewClusterBox creates a ClusterBox of the given size
func NewClusterBox(size int, newNode NewNodeFunc) (
	*ClusterBox, context.CancelFunc, error) {
	nodes := make([]Node, 0, size)
	cancels := make([]context.CancelFunc, 0, size)
	ctx, cancel := context.WithCancel(context.Background())
	for i := 0; i < size; i++ {
		nodeCtx, cancelNode := context.WithCancel(ctx)
		node, err := newNode(nodeCtx, i)
		if err != nil {
			cancelNode()
			return nil, cancel, err
		}
		nodes = append(nodes, node)
		cancels = append(cancels, cancelNode)
		fmt.Printf("Node %d listens at %s\n", i, node.Endpoint())
	}
	return &ClusterBox{nodes: nodes, cancels: cancels, ctx: ctx}, cancel, nil
}

func (cb *ClusterBox) endpoints() []string {
	var endpoints []string
	for _, node := range cb.nodes {
		endpoints = append(endpoints, node.Endpoint())
	}
	return endpoints
}

// Run the ClusterBox until all nodes stop
func (cb *ClusterBox) Run() {
	endpoints := cb.endpoints()
	for i, n := range cb.nodes {
		cb.wg.Add(1)
		go func(n Node, cancel context.CancelFunc) {
			// Node setup
			n.Setup(endpoints)
			// Launch the server side
			serverDone := make(chan struct{})
			go func(n Node, serverDone chan struct{}) {
				n.Serve()
				fmt.Printf("%s server is done\n", n.Endpoint())
				close(serverDone)
			}(n, serverDone)
			// Luanch the client side
			clientDone := make(chan struct{})
			go func(n Node, clientDone chan struct{}) {
				n.Client()
				fmt.Printf("%s client is done\n", n.Endpoint())
				close(clientDone)
			}(n, clientDone)
			// If client or server are done, cancel the whole node
			var otherDone chan struct{}
			select {
			case <-serverDone:
				otherDone = clientDone
				fmt.Printf("%s waits for client to close...\n", n.Endpoint())
			case <-clientDone:
				otherDone = serverDone
				fmt.Printf("%s waits for server to close...\n", n.Endpoint())
			}
			cancel()
			<-otherDone
			fmt.Printf("%s closed both client&server\n", n.Endpoint())
			cb.wg.Done()
		}(n, cb.cancels[i])
	}
	cb.wg.Wait()
}

// CancelByCtrlC will trigger cancel on Ctrl+c
func CancelByCtrlC(cancel context.CancelFunc) {
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	go func() {
		<-interrupts
		fmt.Println("Received Ctrl+C, cancelling clusterbox...")
		cancel()
	}()
}

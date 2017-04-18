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
	nodes []Node
	wg    sync.WaitGroup
	ctx   context.Context
}

// NewClusterBox creates a ClusterBox of the given size
func NewClusterBox(size int, newNode NewNodeFunc) (
	*ClusterBox, context.CancelFunc, error) {
	nodes := make([]Node, 0, size)
	for i := 0; i < size; i++ {
		node, err := newNode(i)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, node)
		fmt.Printf("Node %d listens at %s\n", i, node.Endpoint())
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ClusterBox{nodes: nodes, ctx: ctx}, cancel, nil
}

// Run the ClusterBox until all nodes stop
func (cb *ClusterBox) Run() {
	go func() {
		<-cb.ctx.Done()
		for _, n := range cb.nodes {
			n.Stop()
		}
	}()
	for _, n := range cb.nodes {
		cb.wg.Add(1)
		go func(n Node) {
			n.Setup(cb.nodes)
			serverDone := make(chan struct{})
			go func(n Node, serverDone chan struct{}) {
				n.Serve()
				fmt.Printf("%s server is done\n", n.Endpoint())
				close(serverDone)
			}(n, serverDone)
			clientDone := make(chan struct{})
			go func(n Node, clientDone chan struct{}) {
				n.Client()
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
			cb.wg.Done()
		}(n)
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

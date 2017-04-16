package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
)

func dieOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
		os.Exit(-1)
	}
}

type clusterbox struct {
	nodes []*node
	wg    sync.WaitGroup
}

func newNode(i int) (*node, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	return &node{
		ln:       ln,
		nodeList: make([]*nodeRef, 0),
	}, nil
}

func newClusterBox(maxNodes int) *clusterbox {
	nodes := make([]*node, 0, maxNodes)
	for i := 0; i < maxNodes; i++ {
		node, err := newNode(i)
		dieOnError(err)
		nodes = append(nodes, node)
		fmt.Printf("Node %d listens at %s\n", i, node.endpoint())
	}
	return &clusterbox{nodes: nodes}
}

func (c *clusterbox) run() {
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
		go func(ctx context.Context, n *node) {
			nodeCtx, cancelNode := context.WithCancel(ctx)
			n.setup(nodeCtx, cancelNode, c.nodes)
			serverDone := make(chan struct{})
			go func(n *node, serverDone chan struct{}) {
				n.serve()
				fmt.Printf("%s server is done\n", n.endpoint())
				close(serverDone)
			}(n, serverDone)
			clientDone := make(chan struct{})
			go func(n *node, clientDone chan struct{}) {
				n.clientLoop()
				fmt.Printf("%s client is done\n", n.endpoint())
				close(clientDone)
			}(n, clientDone)
			var peerDone chan struct{}
			select {
			case <-serverDone:
				peerDone = clientDone
				fmt.Printf("%s waits for client to close...\n", n.endpoint())
			case <-clientDone:
				peerDone = serverDone
				fmt.Printf("%s waits for server to close...\n", n.endpoint())
			}
			n.stop()
			<-peerDone
			fmt.Printf("%s closed both client&server\n", n.endpoint())
			c.wg.Done()
			cancelNode()
		}(ctx, n)
	}
	c.wg.Wait()
	cancel()
}

func (c *clusterbox) close() {
	for i, node := range c.nodes {
		dieOnError(node.stop())
		fmt.Printf("Stopped node %d listening at %s\n", i, node.endpoint())
	}
}

func runCluster(size int) {
	clusterbox := newClusterBox(size)
	clusterbox.run()
	clusterbox.close()
}

func main() {
	var size int
	flag.IntVar(&size, "Size", 100, "Number of nodes to generate")
	flag.Parse()

	fmt.Printf("Building clusterbox of %d nodes...\n", size)
	runCluster(size)
}

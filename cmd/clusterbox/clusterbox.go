package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
)

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
	for _, n := range c.nodes {
		c.wg.Add(1)
		go func(n *node) {
			n.setup(c.nodes)
			serverDone := make(chan struct{})
			go func(n *node, serverDone chan struct{}) {
				n.serve()
				close(serverDone)
			}(n, serverDone)
			clientDone := make(chan struct{})
			go func(n *node, clientDone chan struct{}) {
				n.clientLoop()
				close(clientDone)
			}(n, clientDone)
			select {
			case <-serverDone:
			case <-clientDone:
			}
			c.wg.Done()
		}(n)
	}
	c.wg.Wait()
}

func (c *clusterbox) close() {
	for i, node := range c.nodes {
		fmt.Printf("Closing node %d listening at %s\n", i, node.endpoint())
		dieOnError(node.close())
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

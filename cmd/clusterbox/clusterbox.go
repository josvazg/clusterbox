package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

type node struct {
	ln net.Listener
}

var nodes []*node

func dieOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
	}
}

func (n *node) endpoint() string {
	return n.ln.Addr().String()
}

func (n *node) close() error {
	return n.ln.Close()
}

type clusterbox struct {
	nodes []*node
}

func newNode(i int) (*node, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	return &node{ln: ln}, nil
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

func (c *clusterbox) close() {
	for i, node := range c.nodes {
		fmt.Printf("Closing node %d listening at %s\n", i, node.endpoint())
		dieOnError(node.close())
	}
}

func runCluster(size int) {
	clusterbox := newClusterBox(size)
	// TODO: initiate some traffic on cluster
	clusterbox.close()
}

func main() {
	var size int
	flag.IntVar(&size, "Size", 100, "Number of nodes to generate")
	flag.Parse()

	fmt.Printf("Building clusterbox of %d nodes...\n", nodes)
	runCluster(size)
}

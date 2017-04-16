package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type nodeRef struct {
	endpoint string
	modified time.Time
}

type node struct {
	ln     net.Listener
	ctx    context.Context
	cancel context.CancelFunc
	// sample specific
	mtx       sync.RWMutex
	nodeList  []*nodeRef
	neighbors int
	stopped   bool
}

var nodes []*node

func (n *node) endpoint() string {
	return n.ln.Addr().String()
}

func (n *node) setup(ctx context.Context, cancel context.CancelFunc, nodes []*node) {
	n.ctx = ctx
	n.cancel = cancel
	n.add(n.endpoint())
	for i, node := range nodes {
		if node == n {
			next := (i + 1) % len(nodes)
			nextNext := (i + 2) % len(nodes)
			n.add(nodes[next].endpoint())
			n.add(nodes[nextNext].endpoint())
		}
	}
	n.neighbors = len(n.nodeList)
	//fmt.Printf("Node %s neightbours are %v\n", n.endpoint(), n.neighbors)
}

func (n *node) add(endpoint string) {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.nodeList = append(n.nodeList, &nodeRef{endpoint, time.Now()})
}

func (n *node) has(endpoint string) bool {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	for _, eRef := range n.nodeList {
		if endpoint == eRef.endpoint {
			return true
		}
	}
	return false
}

func (n *node) incNeighbors() {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.neighbors++
}

func (n *node) size() int {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	return len(n.nodeList)
}

// go to next neighbor, never 0, cause that is this node
func (n *node) next(neighbor int) (int, *nodeRef) {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	neighbor++
	if neighbor >= n.neighbors {
		neighbor = 1
	}
	return neighbor, n.nodeList[neighbor]
}

func (n *node) dumpEndpoints(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
	for _, ref := range n.nodeList {
		fmt.Fprintf(resp, "%s\n", ref.endpoint)
	}
}

func (n *node) serve() {
	//fmt.Printf("%s doing serve()\n", n.endpoint())
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", n.dumpEndpoints)
	endpoint := n.endpoint()
	err := http.Serve(n.ln, serveMux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s serve() died with: %v\n", endpoint, err)
	}
}

func (n *node) clientLoop() {
	neighbor := 0
	client := &http.Client{}
	var pause time.Duration
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-time.After(pause * time.Millisecond):
			neighborIndex, neighborRef := n.next(neighbor)
			neighbor = neighborIndex
			// request endpoint dump from neighbor
			url := "http://" + neighborRef.endpoint
			resp, err := client.Get(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s client() got error: %v\n", n.endpoint(), err)
			}
			// merge endpoints received
			inEndpoints := make([]string, 0, 1)
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				endpoint := strings.Trim(scanner.Text(), " ")
				// TODO validate endpoint
				inEndpoints = append(inEndpoints, endpoint)
			}
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading server reply: %v\n", err)
				return
			}
			sizeBefore := n.size()
			for _, endpoint := range inEndpoints {
				if !n.has(endpoint) {
					n.add(endpoint)
				}
			}
			sizeAfter := n.size()
			if sizeAfter > sizeBefore {
				n.incNeighbors()
				pause = 20
			}
			if sizeAfter != sizeBefore {
				fmt.Printf("%s now knows %d nodes\n", n.endpoint(), sizeAfter)
			} else {
				pause = 500
			}
		}
	}
}

func (n *node) stop() error {
	if !n.stopped {
		n.stopped = true
		n.cancel()
		return n.ln.Close()
	}
	return nil
}

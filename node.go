package clusterbox

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

// ClientFunc is the function handling the Node's client side behavior
type ClientFunc func(context.Context)

// HTTPNode is the generic HTTPNode
type HTTPNode struct {
	ln     net.Listener
	ctx    context.Context
	cancel context.CancelFunc
	client ClientFunc
}

// gossipNode is the sample gossipNode
type gossipNode struct {
	HTTPNode
	mtx       sync.RWMutex
	nodeList  []string
	neighbors int
}

// NewGossipNode creates a new sample gossipNode
func NewGossipNode(i int) (Node, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	node := &gossipNode{
		HTTPNode: HTTPNode{
			ln:     ln,
			ctx:    ctx,
			cancel: cancel,
		},
		nodeList: make([]string, 0),
	}
	node.client = n.clientFn
	return node, nil
}

// Endpoint returns this HTTPNode's endpoint
func (n *HTTPNode) Endpoint() string {
	return n.ln.Addr().String()
}

// Setup prepares up a gossipNode to do work
func (n *gossipNode) Setup(nodes []Node) {
	n.add(n.Endpoint())
	for i, node := range nodes {
		if node.Endpoint() == n.Endpoint() {
			next := (i + 1) % len(nodes)
			nextNext := (i + 2) % len(nodes)
			n.add(nodes[next].Endpoint())
			n.add(nodes[nextNext].Endpoint())
		}
	}
	n.neighbors = len(n.nodeList)
	//fmt.Printf("Node %s neightbours are %v\n", n.endpoint(), n.neighbors)
}

func (n *gossipNode) add(endpoint string) {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.nodeList = append(n.nodeList, endpoint)
}

func (n *gossipNode) has(endpoint string) bool {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	for _, ep := range n.nodeList {
		if endpoint == ep {
			return true
		}
	}
	return false
}

func (n *gossipNode) incNeighbors() {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	n.neighbors++
}

func (n *gossipNode) size() int {
	n.mtx.Lock()
	defer n.mtx.Unlock()
	return len(n.nodeList)
}

// go to next neighbor, never 0, cause that is this node
func (n *gossipNode) next(neighbor int) (int, string) {
	n.mtx.RLock()
	defer n.mtx.RUnlock()
	neighbor++
	if neighbor >= n.neighbors {
		neighbor = 1
	}
	return neighbor, n.nodeList[neighbor]
}

func (n *gossipNode) dumpEndpoints(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
	for _, endpoint := range n.nodeList {
		fmt.Fprintf(resp, "%s\n", endpoint)
	}
}

func (n *gossipNode) Serve() {
	//fmt.Printf("%s doing serve()\n", n.endpoint())
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", n.dumpEndpoints)
	endpoint := n.Endpoint()
	err := http.Serve(n.ln, serveMux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s serve() died with: %v\n", endpoint, err)
	}
}

// ClientLoop runs the client until completed or the node stops
func (n *HTTPNode) ClientLoop() {
	n.client(n.ctx)
}

func (n *gossipNode) clientFn(ctx context.Context) {
	neighbor := 0
	client := &http.Client{}
	var pause time.Duration
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(pause * time.Millisecond):
			neighborIndex, endpoint := n.next(neighbor)
			neighbor = neighborIndex
			// request endpoint dump from neighbor
			url := "http://" + endpoint
			resp, err := client.Get(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s client() got error: %v\n", n.Endpoint(), err)
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
				fmt.Printf("%s now knows %d nodes\n", n.Endpoint(), sizeAfter)
			} else {
				pause = 500
			}
		}
	}
}

// Stop the HTTPNode work (Serve & ClientLoop)
func (n *HTTPNode) Stop() error {
	select {
	case <-n.ctx.Done():
		// when ctx is already closed
	default:
		// when ctx is not closed yet
		n.cancel()
	}
	return n.ln.Close()
}

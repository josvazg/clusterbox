package clusterbox

import (
	"bufio"
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
	ln        net.Listener
	endClient chan struct{}
	// sample specific
	mtx       sync.RWMutex
	nodeList  []*nodeRef
	neighbors int
}

// NewNode creates a new sample node
func NewNode(i int) (Node, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	return &node{
		ln:        ln,
		endClient: make(chan struct{}),
		nodeList:  make([]*nodeRef, 0),
	}, nil
}

func (n *node) Endpoint() string {
	return n.ln.Addr().String()
}

func (n *node) Setup(nodes []Node) {
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

func (n *node) Serve() {
	//fmt.Printf("%s doing serve()\n", n.endpoint())
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", n.dumpEndpoints)
	endpoint := n.Endpoint()
	err := http.Serve(n.ln, serveMux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s serve() died with: %v\n", endpoint, err)
	}
}

func (n *node) ClientLoop() {
	neighbor := 0
	client := &http.Client{}
	var pause time.Duration
	for {
		select {
		case <-n.endClient:
			return
		case <-time.After(pause * time.Millisecond):
			neighborIndex, neighborRef := n.next(neighbor)
			neighbor = neighborIndex
			// request endpoint dump from neighbor
			url := "http://" + neighborRef.endpoint
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

func (n *node) Stop() error {
	select {
	case <-n.endClient:
	default:
		close(n.endClient)
	}
	return n.ln.Close()
}

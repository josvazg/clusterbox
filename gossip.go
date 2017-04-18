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

// GossipNode is a full sample Node implementation
type GossipNode struct {
	HTTPNode
	mtx       sync.RWMutex
	nodeList  []string
	neighbors int
}

// NewGossipNode is a NewNodeFunc creating a GossipNode from an HTTPNode
func NewGossipNode(cctx context.Context, i int) (Node, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return nil, err
	}
	gh := &GossipNode{
		HTTPNode: HTTPNode{ln: ln, cctx: cctx},
		nodeList: make([]string, 0),
	}
	return gh, nil
}

// Setup prepares a GossipNode to do work
func (gn *GossipNode) Setup(endpoints []string) {
	gn.add(gn.Endpoint())
	for i, endpoint := range endpoints {
		if endpoint == gn.Endpoint() {
			next := (i + 1) % len(endpoints)
			nextNext := (i + 2) % len(endpoints)
			gn.add(endpoints[next])
			gn.add(endpoints[nextNext])
		}
	}
	gn.neighbors = len(gn.nodeList)
	//fmt.Printf("Node %s neightbours are %v\n", n.endpoint(), n.neighbors)
}

// Serve runs a GossipNode server side
//
// It is setup to dump all know node endpoints to any request
func (gn *GossipNode) Serve() {
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", gn.dumpEndpoints)
	gn.HTTPNode.ServeWith(serveMux)
}

// Client runs all GossipNode's client actions
func (gn *GossipNode) Client() {
	neighbor := 0
	client := &http.Client{}
	var pause time.Duration
loop:
	for {
		select {
		case <-gn.cctx.Done():
			break loop
		case <-time.After(pause * time.Millisecond):
			neighborIndex, endpoint := gn.next(neighbor)
			neighbor = neighborIndex
			// request endpoint dump from neighbor
			url := "http://" + endpoint
			resp, err := client.Get(url)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s client() got error: %v\n", gn.Endpoint(), err)
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
			sizeBefore := gn.size()
			for _, endpoint := range inEndpoints {
				if !gn.has(endpoint) {
					gn.add(endpoint)
				}
			}
			sizeAfter := gn.size()
			if sizeAfter > sizeBefore {
				gn.incNeighbors()
				pause = 20
			}
			if sizeAfter != sizeBefore {
				fmt.Printf("%s now knows %d nodes\n", gn.Endpoint(), sizeAfter)
			} else {
				pause = 500
			}
		}
	}
	fmt.Printf("%s client exists knowing %d nodes\n", gn.Endpoint(), gn.size())
	gn.ln.Close()
}

func (gn *GossipNode) add(endpoint string) {
	gn.mtx.Lock()
	defer gn.mtx.Unlock()
	gn.nodeList = append(gn.nodeList, endpoint)
}

func (gn *GossipNode) has(endpoint string) bool {
	gn.mtx.RLock()
	defer gn.mtx.RUnlock()
	for _, ep := range gn.nodeList {
		if endpoint == ep {
			return true
		}
	}
	return false
}

func (gn *GossipNode) incNeighbors() {
	gn.mtx.Lock()
	defer gn.mtx.Unlock()
	gn.neighbors++
}

func (gn *GossipNode) size() int {
	gn.mtx.Lock()
	defer gn.mtx.Unlock()
	return len(gn.nodeList)
}

// go to next neighbor, never 0, cause that is this node
func (gn *GossipNode) next(neighbor int) (int, string) {
	gn.mtx.RLock()
	defer gn.mtx.RUnlock()
	neighbor++
	if neighbor >= gn.neighbors {
		neighbor = 1
	}
	return neighbor, gn.nodeList[neighbor]
}

func (gn *GossipNode) dumpEndpoints(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)
	for _, endpoint := range gn.nodeList {
		fmt.Fprintf(resp, "%s\n", endpoint)
	}
}

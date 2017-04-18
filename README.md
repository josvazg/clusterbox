# ClusterBox
Run a cluster transport or protocol test in a single box

When designing cluster protocols or testign the impact of some cluster wide changes, sometimes you wish you could do it all in a single box, easily fast & cheap... You are lucky, clusterbox does just that for you, so long you are doing it in Golang.

Mind you, clusterbox might not be suitable for all your testing and validation needs, for instance, you can't expect to do performance tests or run too many heavy loaded nodes in a single box. But, having said that, you can probably:
* Prototype and test cluster protocols (eg. consensus protocols like Paxos or Raft)
* Run a lightweight version of prototype of your service, or a subset of it, with light load.
* Experiment a transport modification impact by comparing clusterboxes with and without the changes.

I wrote this initially for the latest use case. I plan a Mutual TLS project and i would like to compare behaviour and performance of a cluster with Mutual hot-rotating TLS certificates vs the same with plain sockets or static certificates.

## Usage

### Running a ClusterBox

See [cmd/clusterbox/clusterbox.go](http://github.com/josvazg/clusterbox/blob/master/cmd/clusterbox/clusterbox.go):

```go
	cb, cancel, err := clusterbox.NewClusterBox(size, NewNodeFunc)
	...
	clusterbox.CancelByCtrlC(cancel)
	cb.Run()
```

The ```NewClusterBox()``` function will create a ClusteBox object for you the the given size and using the function ```NewNodeFunc()``` to populate its nodes. All the rest of the customization lies in the implementation of those Nodes.

Optionally you can call ```clusterbox.CancelByCtrlC(cancel)``` to have the running Clustebox be gracefully terminated upon pressing *Ctrl+C*, or you can use the cancel function from some other goroutine as you see fit.

Finally you call the clusterbox object's ```Run()``` method to have it run until completed or cancelled by ```cancel()```.

### Creating Nodes

As said before ClusterBox can only run Nodes as created by ```NewNodeFunc```:

```go
type NewNodeFunc func(context.Context, int) (Node, error)
```

The new nodes to be created take:
* A cancellable context that the node needs to use to know when it has been cancelled and thus have to stop work and exit gracefully.
* The node number in the cluster, in case the node to be generated may be different in configuration or specification and the node number can be used as a hint for deciding on the different options.

### Implementing a Node

```Node``` is a go interface defined at [node.go](https://github.com/josvazg/clusterbox/blob/master/node.go) with the following methods:
```go
	Endpoint() string
	Setup(endpoints []string)
	Serve()
	Client()
```
* ```Endpoint()``` returns the URL at which the Node's server side is listening.
* ```Setup(endpoints []string)``` prepares the node to do work.
  * It receives as parameter the complete list of all nodes endpoints, as it is VERY likely that a node might need to know at least some of the other node member's endpoints so that it can communicate with them.
* ```Serve()``` is the blocking server side loop of the node, just like ```http.Serve()```.
  * Must return ONLY when the Node is done doing its work, the node's context was cancelled or the Node had a fatal failure.
* ```Client()``` is the blocking client side loop of the node.
  * Must also return ONLY on completion, node'es cancellatiuon or failure.

#### Caveats
* **Do NOT leave Serve() or Client() implementations empty**: Even if your node is just a pure Client or a Server, return ONLY after the Node work is done or the Node's context given at construction is cancelled. Otherwise **ClusterBox** WILL see the empty side ended and interpret it must cancel the whole node, closing the other *active* side as well.
* **Do NOT use http convenience functions or default global variables**, such as *http.DefaultServeMux*: All nodes will run without isolation in the same process space, so keep them isolated by NOT sharing variables between them.
  
## Sample Node & Cluster Protocol
  
A full node sample implementation can be seen at [gossip.go](https://github.com/josvazg/clusterbox/blob/master/gossip.go). It is a simple gossip protocol that:
* Sets up each node to know the two nodes after it (rolling to the first & second for the last & second to last nodes)
* The server side returns the current know list of node endpoints.
* The client side queries know endpoints (servers) and adds to the known list of endpoints any new/unknown node found in the reply.
The expectative is that, after a few iterations, all nodes should now all others.

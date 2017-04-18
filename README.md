# ClusterBox
Run a cluster transport or protocol test in a single box

When designing cluster protocols or testign the impact of some cluster wide changes, sometimes you wish you could do it all in a single box, easily fast & cheap... You are lucky, clusterbox does just that for you, so long you are doing it in Golang.

Mind you, clusterbox might not be suitable for all your testing and validation needs, for instance, you can't expect to do performance tests or run too many heavy loaded nodes in a single box. But, having said that, you can probably:
* Prototype and test cluster protocols (eg. consensus protocols like Paxos or Raft)
* Run a lightweight version of prototype of your service, or a subset of it, with light load.
* Experiment a transport modification impact by comparing clusterboxes with and without the changes.

I wrote this initially for the latest use case. I plan a Mutual TLS project and i would like to compare behaviour and performance of a cluster with Mutual hot-rotating TLS certificates vs the same with plain sockets or static certificates.

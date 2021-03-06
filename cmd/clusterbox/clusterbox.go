package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/josvazg/clusterbox"
)

func dieOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
		os.Exit(-1)
	}
}

func runCluster(size int) {
	cb, cancel, err := clusterbox.NewClusterBox(
		size, clusterbox.NewGossipNode)
	dieOnError(err)
	clusterbox.CancelByCtrlC(cancel)
	cb.Run()
}

func main() {
	var size int
	flag.IntVar(&size, "Size", 100, "Number of nodes to generate")
	flag.Parse()

	fmt.Printf("Building clusterbox of %d nodes...\n", size)
	runCluster(size)
}

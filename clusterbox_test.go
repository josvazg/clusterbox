package clusterbox_test

import (
	"testing"

	"github.com/josvazg/clusterbox"
)

func dieOnError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

type EmptyNode struct{}

func (en *EmptyNode) Endpoint() string {
	return "empty"
}

func (en *EmptyNode) Setup(nodes []clusterbox.Node) {}

func (en *EmptyNode) Serve() {}

func (en *EmptyNode) Client() {}

func (en *EmptyNode) Stop() error {
	return nil
}

func NewEmptyNode(i int) (clusterbox.Node, error) {
	return &EmptyNode{}, nil
}

var sizes = []int{1, 10, 100}

func TestClusterBoxWithEmptyNode(t *testing.T) {
	for _, size := range sizes {
		clusterbox, _, err := clusterbox.NewClusterBox(size, NewEmptyNode)
		dieOnError(t, err)
		clusterbox.Run()
	}
}

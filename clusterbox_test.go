package clusterbox_test

import (
	"context"
	"io/ioutil"
	"log"
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

func (*EmptyNode) Setup(endpoints []string) {}
func (*EmptyNode) Serve()                   {}
func (*EmptyNode) Client()                  {}
func (*EmptyNode) Stop() error              { return nil }

func NewEmptyNode(cctx context.Context, i int) (clusterbox.Node, error) {
	return &EmptyNode{}, nil
}

var sizes = []int{1, 10, 100}

func setup() {
	log.SetOutput(ioutil.Discard)
}

func TestClusterBoxWithEmptyNode(t *testing.T) {
	setup()
	for _, size := range sizes {
		clusterbox, _, err := clusterbox.NewClusterBox(size, NewEmptyNode)
		dieOnError(t, err)
		clusterbox.Run()
	}
}

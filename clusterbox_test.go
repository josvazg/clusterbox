package clusterbox_test

import (
	"context"
	"io/ioutil"
	"log"
	"testing"
	"time"

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

func NewEmptyNode(cctx context.Context, i int) (clusterbox.Node, error) {
	return &EmptyNode{}, nil
}

func NewTCP4IdleNode(cctx context.Context, i int) (clusterbox.Node, error) {
	return clusterbox.NewIdleNode(cctx, "tcp4")
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

func timeout(t *testing.T, duration time.Duration) *time.Timer {
	return time.AfterFunc(5*time.Second, func() {
		t.Fatal("Timeout!")
	})
}

func TestClusterBoxWithIdleNode(t *testing.T) {
	setup()
	for _, size := range sizes {
		cancelled := false
		clusterbox, cancel, err := clusterbox.NewClusterBox(
			size, NewTCP4IdleNode)
		dieOnError(t, err)
		timeout := timeout(t, 5*time.Second)
		defer timeout.Stop()
		timer := time.AfterFunc(500*time.Millisecond, func() {
			cancel()
			cancelled = true
		})
		defer timer.Stop()
		clusterbox.Run()
		if !cancelled {
			t.Fatalf("IdleNode was expected to have been cancelled!")
		}
	}
}

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func dieOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}
}

func main() {
	url := ""
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Got %d args, expected 2:\n", len(os.Args))
		fmt.Fprintf(os.Stderr, "Usage: %s {url}\n", os.Args[0])
		os.Exit(-1)
	}
	url = os.Args[1]
	if !strings.HasPrefix(url, "http://") {
		url = "http://" + url
	}
	rsp, err := http.Post(url, "text/plain", bytes.NewBufferString("hi!"))
	dieOnError(err)
	io.Copy(os.Stdout, rsp.Body)
}

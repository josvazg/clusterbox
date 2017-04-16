package main

import (
	"io"
	"net/http"
	"fmt"
)

func echoHandler(rsp http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(rsp, "You said:\n")
	io.Copy(rsp, r.Body)
}

func main() {
	http.HandleFunc("/", echoHandler)
	http.ListenAndServe(":7007", nil)
}

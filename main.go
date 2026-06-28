//go:build !local

package main

import (
	"fmt"
	"net"

	"github.com/peiblow/vvm/vm"
)

func main() {
	runtime := vm.NewRuntime()

	ln, err := net.Listen("tcp", ":8332")
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer ln.Close()

	fmt.Println("VVM Runtime listening on :8332")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go runtime.HandleConnection(conn)
	}
}

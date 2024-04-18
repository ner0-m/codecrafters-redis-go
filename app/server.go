package main

import (
	"fmt"
	"net"
	"os"
)

func handleClient(conn net.Conn) {
	defer conn.Close()

	b := []byte("+PONG\r\n")
	_, err = conn.Write(b)

	if err != nil {
		fmt.Println("Error writing to connection: ", err.Error())
		return
	}
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		handleClient(conn)

	}
}

package main

import (
	"fmt"
	"io"
	"net"
	"os"
)

func handler(conn net.Conn) {
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)

		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed")
				return
			}

			fmt.Println("Error reading: ", err.Error())
			return
		}

		// Get the request
		// fmt.Printf("Received: %s", buf[:n])

		_, err = conn.Write([]byte("+PONG\r\n"))
		if err != nil {
			fmt.Println("Error writing: ", err.Error())
			return
		}
	}
}

func eventLoop(connections chan net.Conn) {
	for conn := range connections {
		fmt.Println("New connection")
		go handler(conn)
	}
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server is listening on port 6379")

	connections := make(chan net.Conn)
	go eventLoop(connections)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

        connections <- conn
	}
}

package main

import (
    "io"
    "log"
	"fmt"
	"net"
	"os"
)

func handler(conn net.Conn) {
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)

		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed")
				return
			}

			fmt.Println("Error reading: ", err.Error())
			return
		}

		// Get the request
		log.Printf("Received: %s", buf[:n])
		// See if the request is a PING
		_, err = conn.Write([]byte("+PONG\r\n"))
		if err != nil {
			fmt.Println("Error writing: ", err.Error())
			return
		}
	}

	//
	// b := []byte("+PONG\r\n")
	// _, err := conn.Write(b)
	//
	// if err != nil {
	// 	fmt.Println("Error writing to connection: ", err.Error())
	// 	return
	// }
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server is listening on port 6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		fmt.Println("Accepted request")

		handler(conn)
	}
}

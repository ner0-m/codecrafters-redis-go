package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func handleCommand(resp Response, store Store) ([]byte, error) {
	str := resp.String()
	lines := strings.Split(str, "\r\n")

	var cmd []string
	for _, line := range lines {
		if len(line) == 0 || line[0] == '$' {
			continue
		}
		cmd = append(cmd, line)
	}

	switch strings.ToLower(cmd[0]) {
	case "ping":
		fmt.Printf("Debug: echo\n")
		return ping()
	case "echo":
		fmt.Printf("Debug: echo %s\n", cmd[1])
		return echo(cmd[1])
	case "set":
		return set(cmd, store)
	case "get":
        return get(cmd[1], store)
    case "info":
        if len(cmd) == 2 {
            return info(cmd[1])
        } else {
            return info("")
        }
	}
	return nil, errors.New("Unknown command: '" + strings.Join(cmd[:], " ") + "'")
}

func parseMsg(msg []byte, store Store) ([]byte, error) {
	s, resp := ReadNextRESP(msg)

	if s == 0 {
		return nil, errors.New("no resp object")
	}

	t := resp.Type

	switch t {
	case Int:
	case Status:
	case Bulk:
	case Array:
		response, err := handleCommand(resp, store)
		if err != nil {
			return nil, err
		}

		return response, nil
	case Error:
		return []byte(""), nil
	}
	return nil, nil
}

func handler(conn net.Conn, store Store) {
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

		response, err := parseMsg(buf[:n], store)
		if err != nil {
			fmt.Println("Error reading resp", err.Error())
			os.Exit(1)
		}

		if response != nil {
			_, err = conn.Write(response)

			if err != nil {
				fmt.Println("Error writing: ", err.Error())
				os.Exit(1)
			}
		}
	}
}

func eventLoop(connections chan net.Conn, store Store) {
	for conn := range connections {
		fmt.Println("New connection")
		go handler(conn, store)
	}
}

func main() {
	port := flag.Int("port", 6379, "port to start TCP server on")
	flag.Parse()

	store := Store{
		Store: make(map[string]Value),
	}

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		fmt.Printf("Failed to bind to port %d\n", *port)
		os.Exit(1)
	}
	defer l.Close()

	fmt.Printf("Server is listening on port %d\n", *port)

	connections := make(chan net.Conn)
	go eventLoop(connections, store)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		connections <- conn
	}
}

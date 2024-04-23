package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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
		return []byte("+PONG\r\n"), nil
	case "echo":
		fmt.Printf("Debug: echo %s\n", cmd[1])
		return encodeBulk(cmd[1]), nil
	case "set":
		if len(cmd) == 3 {
			fmt.Printf("Debug: set %s = %s\n", cmd[1], cmd[2])
			store.Write(cmd[1], cmd[2], nil)
			return []byte("+OK\r\n"), nil
		} else if len(cmd) == 5 && strings.ToLower(cmd[3]) == "px" {
			fmt.Printf("Debug: set %s = %s, %s %s\n", cmd[1], cmd[2], cmd[3], cmd[4])
			d, err := strconv.Atoi(cmd[4])
			if err != nil {
				return nil, errors.New("Could not convert expiery length")
			}
			var dur *time.Duration
			dur = new(time.Duration)
			*dur = time.Duration(d) * time.Millisecond
			store.Write(cmd[1], cmd[2], dur)
			return []byte("+OK\r\n"), nil
		}
	case "get":
		v, ok := store.Read(cmd[1])
		if !ok {
			fmt.Printf("Debug: get %s, not in store\n", cmd[1])
			return []byte("$-1\r\n"), nil
		}
		fmt.Printf("Debug: get %s = %s\n", cmd[1], v)
		return encodeBulk(v), nil
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

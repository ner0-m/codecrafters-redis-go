package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	Error  = '-'
	Status = '+'
	Int    = ':'
	Bulk   = '$'
	Array  = '*'
)

var mtx = sync.RWMutex{}

type Type byte

type Response struct {
	Type  Type
	Raw   []byte
	Data  []byte
	Count int
}

func (r Response) Bytes() []byte {
	return r.Data
}

func (r Response) String() string {
	return string(r.Data)
}

func parseInt(data []byte) error {
	if len(data) == 0 {
		return errors.New("Can not parse empty data as Int")
	}
	_, err := strconv.ParseInt(string(data), 10, 64)

	return err
}

func parseBulk(count int, i int, b []byte) error {
	if count < 0 {
		return errors.New("Can not convert to Bulk string of non-positive length")
	}

	if len(b) < i+count+2 {
		return errors.New("Given bytes and length do not match")
	}

	if b[i+count] != '\r' && b[i+count+1] != '\n' {
		return errors.New("String doesn't end with \\r\\n")
	}

	return nil
}

func encodeBulk(str string) []byte {
	return []byte("$" + strconv.Itoa(len(str)) + "\r\n" + str + "\r\n")
}

func ReadNextRESP(b []byte) (n int, resp Response) {
	if len(b) == 0 {
		return 0, Response{}
	}

	resp.Type = Type(b[0])
	switch resp.Type {
	case Error, Status, Int, Bulk, Array:
	default:
		// Invalid Type
		return 0, Response{}
	}
	// Find next \r\n
	i := strings.Index(string(b), "\r\n")

	// Couldn't find it, so it's invalid
	if i == -1 {
		return 0, Response{}
	}

	i += 2 // move after \r\n

	resp.Raw = b[0:i]
	resp.Data = b[1 : i-2]

	if resp.Type == Int {
		err := parseInt(resp.Data)

		if err != nil {
			return 0, Response{}
		}

		return len(resp.Raw), resp
	}

	if resp.Type == Error {
		return len(resp.Raw), resp
	}

	count, err := strconv.Atoi(resp.String())
	if err != nil {
		return 0, Response{}
	}

	if resp.Type == Bulk {
		err = parseBulk(count, i, b)

		if err != nil {
			return 0, Response{}
		}

		resp.Data = b[i : i+count]
		resp.Raw = b[0 : i+count+2]
		return len(resp.Raw), resp
	}

	// It's an array for sure
	var k int
	slice := b[i:]
	for j := 0; j < count; j++ {
		sn, subresp := ReadNextRESP(slice)

		if subresp.Type == 0 {
			return 0, Response{}
		}

		k += sn
		slice = slice[sn:]
	}

	resp.Data = b[i : i+k]
	resp.Raw = b[0 : i+k]
	resp.Count = count

	return len(resp.Raw), resp
}

func handleCommand(resp Response, store map[string]string) ([]byte, error) {
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
		fmt.Printf("Debug: set %s = %s\n", cmd[1], cmd[2])
        mtx.Lock()
		store[cmd[1]] = cmd[2]
        mtx.Unlock()
		return []byte("+OK\r\n"), nil
	case "get":
        mtx.Lock()
		v, ok := store[cmd[1]]
        mtx.Unlock()
		if !ok {
		    fmt.Printf("Debug: get %s, not in store\n", cmd[1])
			return []byte("$-1\r\n"), nil
		}
		fmt.Printf("Debug: get %s = %s\n", cmd[1], v)
		return encodeBulk(v), nil
	}
	return nil, errors.New("Unknown command: '" + strings.Join(cmd[:], " ") + "'")
}

func parseMsg(msg []byte, store map[string]string) ([]byte, error) {
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

func handler(conn net.Conn, store map[string]string) {
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

func eventLoop(connections chan net.Conn, store map[string]string) {
	for conn := range connections {
		fmt.Println("New connection")
		go handler(conn, store)
	}
}

func main() {
	store := make(map[string]string)

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server is listening on port 6379")

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

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
	"sync"
)

func readConn(ch chan []byte, conn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				fmt.Println(conn, "EOF found, closing connection")
				return
			}

			fmt.Println(conn, "Error reading: ", err.Error())
			return
		}

		ch <- buf[0:n]
	}

}

func parseMsg(msg []byte) (Command, []byte, error) {
	s, resp := ReadNextRESP(msg)

	raw := msg[:s]

	if s == 0 {
		return Command{}, msg, errors.New("Message of length 0")
	}

	if s == -1 {
		return Command{}, msg, errors.New("Invalid encoding type")
	} else if s == -1 {
		return Command{}, msg, errors.New("Missing expected \\r\\n")
	}

	if resp.Type == Error {
		return Command{ERROR, make([]string, 0), raw}, msg[s:], nil
	}

	if resp.Type != Int && resp.Type != Status && resp.Type != Bulk && resp.Type != Array {
		return Command{}, msg[s:], errors.New("Unknown Respond Type")
	}

	str := resp.String()
	lines := strings.Split(str, "\r\n")

	var cmds []string
	for _, line := range lines {
		if len(line) == 0 || line[0] == '$' {
			continue
		}
		cmds = append(cmds, line)
	}

	cmd := strings.ToLower(cmds[0])
	args := cmds[1:]

	return Command{
		Type: cmd,
		Args: args,
		Raw:  raw,
	}, msg[s:], nil
}

func processMsg(bufCh chan []byte, cmdCh chan Command) {
	for {
		select {
		case buf := <-bufCh:
			for len(buf) > 0 {
				var cmd Command
				var err error
				cmd, buf, err = parseMsg(buf)

				if err != nil {
					fmt.Println("Error parsing response", err.Error())
					close(cmdCh)
					return
				}

				cmdCh <- cmd
			}
		}
	}
}

func processCmd(cmdCh chan Command, replCmdCh chan Command, respCh chan []byte, instance *Instance) {
	for cmd := range cmdCh {
		replCmdCh <- cmd

		response, err := cmd.CreateRespond(instance)

		if err != nil {
			fmt.Println("Error Processing Command:", err.Error())
		}

		if response != nil {
			respCh <- response
		}
	}
}

func processResponse(respCh chan []byte, conn net.Conn) {
	for resp := range respCh {
		// fmt.Printf("RESPONDING: %s\n", strconv.Quote(string(resp)))
		_, err := conn.Write(resp)

		if err != nil {
			fmt.Printf("Error writing response: %s\n", err.Error())
		}
	}
}

func handleReplCommands(replCmdCh chan Command, conn net.Conn, instance *Instance) {
	for cmd := range replCmdCh {
		// For now assume this may be only send once
		if cmd.Type == PSYNC {
			instance.ReplMutex.Lock()
			instance.Replicas = append(instance.Replicas, conn)
			instance.ReplMutex.Unlock()
		}

		if cmd.Type == SET {
			instance.ReplMutex.Lock()
			for _, c := range instance.Replicas {
				_, err := c.Write(cmd.Raw)
				if err != nil {
					fmt.Printf("Error forwarding to replica: %s", err.Error())
				}
			}
			instance.ReplMutex.Unlock()
		}
	}
}

func eventLoop(connections chan net.Conn, instance *Instance) {
	for conn := range connections {
		readCh := make(chan []byte)
		cmdCh := make(chan Command)
		replCmdCh := make(chan Command)
		respCh := make(chan []byte)
		go readConn(readCh, conn)
		go processMsg(readCh, cmdCh)
		go processCmd(cmdCh, replCmdCh, respCh, instance)
		go handleReplCommands(replCmdCh, conn, instance)
		go processResponse(respCh, conn)
	}
}

type dict map[string]string
type dict_of_dict map[string]dict

type Instance struct {
	Store     Store
	Info      dict_of_dict
	Master    net.Conn
	ReplMutex sync.RWMutex
	Replicas  []net.Conn
}

func syncSlaveToMaster(conn net.Conn, port string) {
	// Step 1: send ping
	_, err := conn.Write([]byte("*1\r\n$4\r\nping\r\n"))
	if err != nil {
		panic(err)
	}

	// Receive pong
	resp := make([]byte, 1024)
	n, err := conn.Read(resp)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Sync to Master: Response to ping: %s\n", strconv.Quote(string(resp[:n])))

	// Step 2: Send REPLCONF listening-port <PORT>
	_, err = conn.Write(encodeArray([]string{"REPLCONF", "listening-port", port}))
	if err != nil {
		panic(err)
	}

	// Receive OK
	n, err = conn.Read(resp)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sync to Master: Response to REPLCONF listening-port <port>: %s\n", strconv.Quote(string(resp[:n])))

	// Send REPLCONF capa psync2
	_, err = conn.Write(encodeArray([]string{"REPLCONF", "capa", "psync2"}))
	if err != nil {
		panic(err)
	}

	// Receive OK
	n, err = conn.Read(resp)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sync to Master: Response to REPLCONF capa psync2: %s\n", strconv.Quote(string(resp[:n])))

	// Step 3: Send PSYNC
	_, err = conn.Write(encodeArray([]string{"PSYNC", "?", "-1"}))
	if err != nil {
		panic(err)
	}

	n, err = conn.Read(resp)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Sync to Master: Response to PSYNC ? -1: %s\n", strconv.Quote(string(resp[:n])))
}

func handleMasterConn(conn net.Conn, instance *Instance) {
	syncSlaveToMaster(conn, instance.Info["replication"]["port"])

	for {
		readCh := make(chan []byte)
		cmdCh := make(chan Command)
		// replCh := make(chan Command)
		// respCh := make(chan []byte)
		go readConn(readCh, conn)
		go processMsg(readCh, cmdCh)

		for cmd := range cmdCh {
			// replCh <- cmd

			_, err := cmd.CreateRespond(instance)

			if err != nil {
				fmt.Println("Error creating responds:", err.Error())
			}
		}
	}
}

func main() {
	host := "0.0.0.0"
	port := "6379"

	instance := Instance{}
	instance.Info = make(dict_of_dict)
	instance.Info["replication"] = make(dict)
	instance.Info["replication"]["role"] = "master"
	instance.Info["replication"]["master_replid"] = "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"
	instance.Info["replication"]["master_repl_offset"] = "0"

	instance.Store = Store{
		Store: make(map[string]Value),
	}

	// Parse flags
	port_arg_pointer := flag.String("port", port, "--port <PORT>")
	primary_host_arg_pointer := flag.String("replicaof", "", "--replicaof <MASTER HOST> <MASTER PORT>")
	flag.Parse()

	if len(*primary_host_arg_pointer) > 0 {
		master_host := (*primary_host_arg_pointer)
		if len(master_host) > 1 {
			master_port := flag.Arg(0)
			if len(master_port) > 0 {
				instance.Info["replication"]["role"] = "slave"
				instance.Info["replication"]["port"] = master_port
				instance.Info["replication"]["host"] = master_host
			}
		}
	}

	if len(*port_arg_pointer) > 0 {
		port = *port_arg_pointer
	}

	// Sync if we are a slave
	if instance.Info["replication"]["role"] == "slave" {
		masterAddr := net.JoinHostPort(instance.Info["replication"]["host"], instance.Info["replication"]["port"])
		masterConn, err := net.Dial("tcp", masterAddr)
		if err != nil {
			panic(err)
		}

		go handleMasterConn(masterConn, &instance)

		instance.Master = masterConn
	}

	// Start server
	l, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		fmt.Printf("Failed to bind to %s:%s\n", host, port)
		os.Exit(1)
	}
	defer l.Close()

	fmt.Printf("Server is listening on port %s\n", port)

	connections := make(chan net.Conn)
	go eventLoop(connections, &instance)

	// if instance.Master != nil {
	// 	connections <- instance.Master
	// }

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		connections <- conn
	}
}

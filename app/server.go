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
)

func parseMsg(msg []byte) (Command, []byte, error) {
	s, resp := ReadNextRESP(msg)

	if s == 0 {
		return Command{}, msg, errors.New("Length 0")
	}

	if s == -1 {
		return Command{}, msg, errors.New("Invalid Type")
	} else if s == -1 {
		return Command{}, msg, errors.New("No \\r\\n")
	}

	if resp.Type == Error {
		return Command{ERROR, make([]string, 0)}, msg[s:], nil
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
	}, msg[s:], nil
}

func handler(conn net.Conn, instance *Instance) {
	defer conn.Close()

	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)

		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF found, closing connection")
				return
			}

			fmt.Println("Error reading: ", err.Error())
			return
		}
		buf = buf[:n]
		fmt.Printf("Message Received: %s\n", strconv.Quote(string(buf)))

		for len(buf) > 0 {
			var cmd Command
			cmd, buf, err = parseMsg(buf)
			if err != nil {
				fmt.Println("Error parsing response", err.Error())
				os.Exit(1)
			}

			response, err := cmd.Respond(*instance)
			fmt.Printf("Message responds: %s\n", strconv.Quote(string(response)))
			if err != nil {
				fmt.Println("Error creating responds:", err.Error())
				os.Exit(1)
			}

			if response != nil {
				_, err = conn.Write(response)

				if err != nil {
					fmt.Println("Error writing: ", err.Error())
					os.Exit(1)
				}
			}

			if cmd.Type == SET {
				for _, rconn := range instance.Replicas {
					_, err = rconn.Write(response)

					if err != nil {
						fmt.Println("Error writing to replica: ", err.Error())
					}
				}
			}

			if cmd.Type == PSYNC {
				instance.Replicas = append(instance.Replicas, conn)
			}
		}
	}
}

func eventLoop(connections chan net.Conn, instance *Instance) {
	for conn := range connections {
		fmt.Println("New connection")
		go handler(conn, instance)
	}
}

type dict map[string]string
type dict_of_dict map[string]dict

type Instance struct {
	Store    Store
	Info     dict_of_dict
	Replicas []net.Conn
}

func syncSlaveToMaster(masterAddr string, port string) {
	conn, err := net.Dial("tcp", masterAddr)
	if err != nil {
		panic(err)
	}

	// Step 1: send ping
	_, err = conn.Write([]byte("*1\r\n$4\r\nping\r\n"))
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
		syncSlaveToMaster(net.JoinHostPort(instance.Info["replication"]["host"], instance.Info["replication"]["port"]), port)
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

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		connections <- conn
	}
}

package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/commands"
	"github.com/codecrafters-io/redis-starter-go/app/encode"
	"github.com/codecrafters-io/redis-starter-go/app/instance"
	"github.com/codecrafters-io/redis-starter-go/app/parser"
)

type ThreadSafeQueue[T any] struct {
	queue []T
	mutex sync.RWMutex
}

func (q *ThreadSafeQueue[T]) Peek() T {
	q.mutex.Lock()
	val := q.queue[0]
	q.mutex.Unlock()

	return val
}

func (q *ThreadSafeQueue[T]) Pop() T {
	q.mutex.Lock()
	val := q.queue[0]
	q.queue = q.queue[1:]
	q.mutex.Unlock()

	return val
}

func (q *ThreadSafeQueue[T]) Push(val T) {
	q.mutex.Lock()
	q.queue = append(q.queue, val)
	q.mutex.Unlock()
}

func (q *ThreadSafeQueue[T]) Len() int {
	q.mutex.Lock()
	len := len(q.queue)
	q.mutex.Unlock()

	return len
}

type Client struct {
	Conn     net.Conn
	MsgQueue ThreadSafeQueue[parser.Message]
	CmdQueue ThreadSafeQueue[commands.Command]
	ReplMode bool
}

func (c *Client) Receive(msg parser.Message) {
	c.MsgQueue.Push(msg)
}

func (c *Client) NumMessages() int {
	return c.MsgQueue.Len()
}

func (c *Client) NumCommands() int {
	return c.CmdQueue.Len()
}

func (c *Client) HandleNextMsg() (int, commands.Command) {
	msg := c.MsgQueue.Pop()

	cmdstr := strings.ToLower(msg.Data[0])

	cmd := commands.CreateCommand(cmdstr, msg.Data[1:])

	if cmd != nil {
		c.CmdQueue.Push(cmd)
		return len(msg.Raw), cmd
	}

	return 0, nil
}

func (c *Client) ExecuteCommand(cmd commands.Command, inst *instance.Instance) []byte {
	var resp []byte
	var err error
	if cmd != nil {
		resp, err = cmd.Execute(inst)

		if err != nil {
			fmt.Printf("Error executing command: %s", err.Error())
			return nil
		}
	}

	// Forward to replicas
	setcmd, ok := cmd.(*commands.SetCommand)
	if ok {
		conns := inst.GetReplicas()
		// replmsg := strings.Join(setcmd.Raw, "\r\n") + "\r\n"
		replmsg := setcmd.Encode()
		if inst.NumReplicas() > 0 {
			fmt.Printf("Sending %s to replicas\n", strconv.Quote(string(replmsg)))
			for _, conn := range conns {
				conn.Write([]byte(replmsg))
			}
		}
	}

	// For some other messages, we still need to do some work, even if we don't respond, or already have a responds
	_, ok = cmd.(*commands.FullsyncCommand)
	if ok {
		for c.NumMessages() < 1 {
		}
		// Drop it, we don't deal with it
		c.MsgQueue.Pop()
	}

	_, ok = cmd.(*commands.PsyncCommand)
	if ok {
		fmt.Printf("Adding replica\n")

		// Add conn to connection
		inst.AddReplica(c.Conn)
	}

	return resp
}

func (c *Client) Process(output chan []byte, inst *instance.Instance) {
	for {
		if c.NumMessages() > 0 {
			i, cmd := c.HandleNextMsg()

			if cmd != nil {
				resp := c.ExecuteCommand(cmd, inst)

				if resp != nil {
					output <- resp
				}

				inst.Offset += i
			}
		}
	}
}

func (c *Client) ProcessMaster(output chan []byte, inst *instance.Instance) {
	for {
		if c.NumMessages() > 0 {
			n, cmd := c.HandleNextMsg()

			fmt.Printf("Receiving message of length %d\n", n)

			if cmd != nil {
				resp := c.ExecuteCommand(cmd, inst)

				replcmd, ok := cmd.(*commands.ReplconfCommand)

				if resp != nil && ok && replcmd.SubCmd == "getack" {
					output <- resp

				}
				inst.Offset += n
			}
		}
	}
}

func asyncRead(conn net.Conn, client *Client) {
	reader := bufio.NewReader(conn)

	for {
		cur, err := reader.ReadString('\n')

		if err != nil {
			fmt.Printf("Error reading from reader: %s\n", err.Error())
			break
		}

		raw, data := parser.ParseMsg(cur, reader)

		fmt.Printf("Receive: %s\n", strconv.Quote(raw))

		var msg parser.Message
		if raw != "" && data != nil {
			msg = parser.Message{Raw: raw, Data: data}
			client.Receive(msg)
		}
	}
}

func asyncWrite(conn net.Conn, output chan []byte) {
	for resp := range output {
		_, err := conn.Write(resp)

		if err != nil {
			fmt.Printf("Error writing output: %s\n", err.Error())
		}
	}
}

func eventLoop(connections chan net.Conn, inst *instance.Instance) {
	for conn := range connections {
		go func() {
			defer conn.Close()

			fmt.Printf("Working on connection %v\n", conn)

			output := make(chan []byte)

			client := Client{Conn: conn}

			go asyncRead(conn, &client)
			go asyncWrite(conn, output)
			client.Process(output, inst)
		}()
	}
}

func handleMaster(conn net.Conn, port string, inst *instance.Instance) {
	client := Client{Conn: conn}
	output := make(chan []byte)

	go asyncRead(conn, &client)
	go asyncWrite(conn, output)

	// Step 1: send ping
	output <- []byte("*1\r\n$4\r\nPING\r\n")

	// Wait for response
	for client.NumMessages() == 0 {
	}

	msg := client.MsgQueue.Pop()

	cmdstr := strings.ToLower(msg.Data[0])
	if cmdstr != "pong" {
		fmt.Printf("Handshake with master failed, expected pong, is %s\n", strconv.Quote(cmdstr))
	}

	// Step 2: Send REPLCONF listening-port <PORT>
	output <- encode.EncodeArray([]string{"REPLCONF", "listening-port", port})

	// Wait for response
	for client.NumMessages() == 0 {
	}

	msg = client.MsgQueue.Pop()
	cmdstr = strings.ToLower(msg.Data[0])
	if cmdstr != "ok" {
		fmt.Printf("Handshake with master failed, expected OK to REPLCONF listening-port %s\n", port)
	}

	output <- encode.EncodeArray([]string{"REPLCONF", "capa", "psync2"})

	// Wait for response
	for client.NumMessages() == 0 {
	}

	msg = client.MsgQueue.Pop()
	cmdstr = strings.ToLower(msg.Data[0])

	if cmdstr != "ok" {
		fmt.Printf("Handshake with master failed, expected OK to REPLCONF capa psync2\n")
	}

	output <- encode.EncodeArray([]string{"PSYNC", "?", "-1"})

	// Wait for response
	for client.NumMessages() == 0 {
	}

	msg = client.MsgQueue.Pop()
	cmdstr = strings.ToLower(msg.Data[0])
	if cmdstr == "fullresync" {
		fmt.Printf("Handshake with master failed, expected FULLRESYNC to REPLCONF capa psync2\n")
	}

	// Wait for RDB file
	for client.NumMessages() == 0 {
	}
	msg = client.MsgQueue.Pop()

	client.ProcessMaster(output, inst)
}

func main() {
	host := "0.0.0.0"
	port := "6379"

	inst := instance.Instance{}
	inst.Info = make(map[string]map[string]string)
	inst.Info["replication"] = make(map[string]string)
	inst.Info["replication"]["role"] = "master"
	inst.Info["replication"]["master_replid"] = "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"
	inst.Info["replication"]["master_repl_offset"] = "0"

	inst.Store = instance.Store{
		Store: make(map[string]instance.Value),
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
				inst.Info["replication"]["role"] = "slave"
				inst.Info["replication"]["port"] = master_port
				inst.Info["replication"]["host"] = master_host
			}
		}
	}

	if len(*port_arg_pointer) > 0 {
		port = *port_arg_pointer
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
	go eventLoop(connections, &inst)

	// Sync if we are a slave
	if inst.Info["replication"]["role"] == "slave" {
		conn, err := net.Dial("tcp", net.JoinHostPort(inst.Info["replication"]["host"], inst.Info["replication"]["port"]))

		if err != nil {
			fmt.Printf("Error connection to master: %s\n", err.Error())
			os.Exit(1)
		}

		go handleMaster(conn, port, &inst)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		connections <- conn
	}
}

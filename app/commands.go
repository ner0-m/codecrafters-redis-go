package main

import (
    "encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	PING     = "ping"
	ECHO     = "echo"
	SET      = "set"
	GET      = "get"
	INFO     = "info"
	REPLCONF = "replconf"
	PSYNC    = "psync"
	ERROR    = "error"
)

type Command struct {
	Type string
	Args []string
}

func makeArr(b []byte) [][]byte {
	return [][]byte{b}
}

func (cmd *Command) Respond(instance Instance) ([][]byte, error) {
	switch cmd.Type {
	case PING:
		return ping()
	case ECHO:
		return echo(cmd.Args[0])
	case SET:
		return set(cmd.Args, instance.Store)
	case GET:
		return get(cmd.Args[0], instance.Store)
	case INFO:
		return info(cmd.Args[0], instance)
	case REPLCONF:
		return replconf(cmd.Args[0], cmd.Args[1:])
	case PSYNC:
		return psync(cmd.Args, instance)
	case ERROR:
		return makeArr([]byte("")), nil
	}
	return nil, errors.New("Unknown Command")
}

func ping() ([][]byte, error) {
	return makeArr([]byte("+PONG\r\n")), nil
}

func echo(str string) ([][]byte, error) {
	return makeArr(encodeBulk(str)), nil
}

func set(args []string, store Store) ([][]byte, error) {
	if len(args) == 2 {
		fmt.Printf("Debug: set %s = %s\n", args[0], args[1])
		store.Write(args[0], args[1], nil)
		return makeArr([]byte("+OK\r\n")), nil
	} else if len(args) == 4 && strings.ToLower(args[2]) == "px" {
		fmt.Printf("Debug: set %s = %s, %s %s\n", args[0], args[1], args[2], args[3])
		d, err := strconv.Atoi(args[3])
		if err != nil {
			return nil, errors.New("Could not convert expiery length")
		}
		var dur *time.Duration
		dur = new(time.Duration)
		*dur = time.Duration(d) * time.Millisecond
		store.Write(args[0], args[1], dur)
		return makeArr([]byte("+OK\r\n")), nil
	} else {
		return nil, errors.New("Unknown command for set")
	}
}

func get(key string, store Store) ([][]byte, error) {
	v, ok := store.Read(key)
	if !ok {
		fmt.Printf("Debug: get %s, not in store\n", key)
		return makeArr([]byte("$-1\r\n")), nil
	}
	fmt.Printf("Debug: get %s = %s\n", key, v)
	return makeArr(encodeBulk(v)), nil
}

func info(section string, instance Instance) ([][]byte, error) {
	if strings.ToLower(section) == "replication" {
		repl := instance.Info["replication"]
		return makeArr(encodeBulk(fmt.Sprintf("# Replication\r\nrole:%s\r\nmaster_replid:%s\r\nmaster_repl_offset:%s\r\n", repl["role"], repl["master_replid"], repl["master_repl_offset"]))), nil
	} else {
		return nil, errors.New("Unknown INFO section: '" + section + "'")
	}
}

func replconf(cmd string, args []string) ([][]byte, error) {
	fmt.Printf("Command: REPLCONF %s %v\n", cmd, args)
	if strings.ToLower(cmd) == "listening-port" {
		return makeArr([]byte("+OK\r\n")), nil
	} else if strings.ToLower(cmd) == "capa" {
		return makeArr([]byte("+OK\r\n")), nil
	}
	return nil, fmt.Errorf("Unknown command to replconf: %s, with args %v", cmd, args)
}

func psync(args []string, instance Instance) ([][]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("PSYNC requires two arguments, given %v", args)
	}

	if args[0] == "?" && args[1] == "-1" {
		repl := instance.Info["replication"]
		replid := repl["master_replid"]
		repl_offset := repl["master_repl_offset"]
		fullsync := []byte(fmt.Sprintf("+FULLRESYNC %s %s\r\n", replid, repl_offset))

		emptyRDB := []byte("524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2")
		body := make([]byte, hex.DecodedLen(len(emptyRDB)))

		_, err := hex.Decode(body, []byte(emptyRDB))
		if err != nil {
			fmt.Printf("failed to decode RDB file: %s\n", err.Error())
		}
        rdbfile := []byte(fmt.Sprintf("$%d\r\n%s", len(body), body))
		return [][]byte{fullsync, rdbfile}, nil
	}

	return nil, fmt.Errorf("Unknown PSYNC args %v", args)
}

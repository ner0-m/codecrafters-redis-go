package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/instance"
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
	OK       = "ok"
)

type Command interface {
	Execute(inst *instance.Instance) ([]byte, error)
}

type ErrorCommand struct {
	Msg string
}

func (cmd *ErrorCommand) Execute(inst *instance.Instance) ([]byte, error) {
	fmt.Printf("Error encountered: %s\n", cmd.Msg)
	return nil, nil
}

type OkCommand struct{}

func (cmd *OkCommand) Execute(inst *instance.Instance) ([]byte, error) {
	return nil, nil
}

type PongCommand struct{}

func (cmd *PongCommand) Execute(inst *instance.Instance) ([]byte, error) {
	return nil, nil
}

type FullsyncCommand struct{}

func (cmd *FullsyncCommand) Execute(inst *instance.Instance) ([]byte, error) {
	return nil, nil
}

func CreateCommand(t string, args []string) Command {
	if t == "ping" {
		return &PingCommand{}
	} else if t == "pong" {
		return &PongCommand{}
	} else if t == "echo" {
		return &EchoCommand{args[0]}
	} else if t == "info" {
		return &InfoCommand{strings.ToLower(args[0])}
	} else if t == "get" {
		return &GetCommand{args[0]}
	} else if t == "set" {
		return &SetCommand{args[0], args[1], args[2:]}
	} else if t == "replconf" {
		return &ReplconfCommand{strings.ToLower(args[0]), strings.ToLower(args[1])}
	} else if t == "psync" {
		return &PsyncCommand{}
	} else if t == "wait" {
		replCnt, _ := strconv.Atoi(args[0])
		timeout, _ := strconv.Atoi(args[1])
		return &WaitCommand{replCnt, timeout}
	}
	return nil
}

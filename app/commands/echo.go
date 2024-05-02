package commands

import (
	"github.com/codecrafters-io/redis-starter-go/app/client"
	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type EchoCommand struct {
	Payload string
}

func (cmd *EchoCommand) Execute(inst *instance.Instance) ([]byte, error) {
	return client.EncodeBulk(cmd.Payload), nil
}

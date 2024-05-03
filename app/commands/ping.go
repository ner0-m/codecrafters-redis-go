package commands

import (
	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type PingCommand struct{}

func (cmd *PingCommand) Execute(inst *instance.Instance) ([]byte, error) {
	if inst.Info["replication"]["role"] == "slave" {
		inst.Offset += 14
	}

	return []byte("+PONG\r\n"), nil
}

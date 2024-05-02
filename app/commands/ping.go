package commands

import (
	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type PingCommand struct {}

func (cmd *PingCommand) Execute(inst *instance.Instance) ([]byte, error) {
	return []byte("+PONG\r\n"), nil
}

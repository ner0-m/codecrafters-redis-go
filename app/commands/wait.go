package commands

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type WaitCommand struct {
	NumReplicas string
	Time        string
}

func (cmd *WaitCommand) Execute(inst *instance.Instance) ([]byte, error) {
	return []byte(fmt.Sprintf(":%d\r\n", inst.NumReplicas())), nil
}

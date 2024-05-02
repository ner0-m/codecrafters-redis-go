package commands

import (
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/client"
	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type GetCommand struct {
	Key string
}

func (cmd *GetCommand) Execute(inst *instance.Instance) ([]byte, error) {
	val, ok := inst.Store.Read(cmd.Key)
	if !ok {
		fmt.Printf("Debug: get %s, not in store\n", cmd.Key)
		return []byte("$-1\r\n"), nil
	}
	fmt.Printf("Debug: get %s = %s\n", cmd.Key, val)
	return client.EncodeBulk(val), nil
}

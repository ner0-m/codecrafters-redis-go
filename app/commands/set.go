package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/encode"
	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type SetCommand struct {
	Key    string
	Value  string
	Params []string
}

func (cmd *SetCommand) Execute(inst *instance.Instance) ([]byte, error) {
	if len(cmd.Params) == 0 {
		fmt.Printf("Debug: set %s = %s\n", cmd.Key, cmd.Value)
		inst.Store.Write(cmd.Key, cmd.Value, nil)
		inst.Offset += cmd.Len()
		return []byte("+OK\r\n"), nil
	} else if len(cmd.Params) == 2 && strings.ToLower(cmd.Params[0]) == "px" {
		fmt.Printf("Debug: set %s = %s, %s %s\n", cmd.Key, cmd.Value, cmd.Params[0], cmd.Params[1])
		d, err := strconv.Atoi(cmd.Params[1])
		if err != nil {
			return nil, fmt.Errorf("Could not convert expiery length %s", cmd.Params[1])
		}
		var dur *time.Duration
		dur = new(time.Duration)
		*dur = time.Duration(d) * time.Millisecond
		inst.Store.Write(cmd.Key, cmd.Value, dur)
		inst.Offset += cmd.Len()
		return []byte("+OK\r\n"), nil
	}

	return nil, fmt.Errorf("Set command: Unknown parameters for set %s", cmd.Params)
}

func (cmd *SetCommand) Len() int {
	return len(cmd.Encode())
}

func (cmd *SetCommand) Encode() []byte {
	if len(cmd.Params) == 0 {
		return encode.EncodeArray([]string{"SET", cmd.Key, cmd.Value})
	} else {
		return encode.EncodeArray([]string{"SET", cmd.Key, cmd.Value, cmd.Params[0], cmd.Params[1]})
	}
}

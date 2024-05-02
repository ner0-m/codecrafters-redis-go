package commands

import (
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/encode"
	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type ReplconfCommand struct {
	SubCmd string
	Arg    string
}

func (cmd *ReplconfCommand) Execute(inst *instance.Instance) ([]byte, error) {
	if strings.ToLower(cmd.SubCmd) == "getack" {
		offset := inst.Offset
		return encode.EncodeArray([]string{"REPLCONF", "ACK", strconv.Itoa(offset)}), nil
	}

	return []byte("+OK\r\n"), nil
}

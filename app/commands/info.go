package commands

import (
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/client"
	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type InfoCommand struct {
	Section string
}

func (cmd *InfoCommand) Execute(inst *instance.Instance) ([]byte, error) {
	if cmd.Section == "replication" {
		repl := inst.Info["replication"]
		str := fmt.Sprintf("# Replication\r\nrole:%s\r\nmaster_replid:%s\r\nmaster_repl_offset:%s\r\n", repl["role"], repl["master_replid"], repl["master_repl_offset"])
		return client.EncodeBulk(str), nil
	}

	return nil, fmt.Errorf("Info Command: Unknown Section %s", cmd.Section)
}

package commands

import (
	"encoding/hex"
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type PsyncCommand struct{}

func (cmd *PsyncCommand) Execute(inst *instance.Instance) ([]byte, error) {
	repl := inst.Info["replication"]
	replid := repl["master_replid"]
	repl_offset := repl["master_repl_offset"]

	emptyRDB := []byte("524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2")
	body := make([]byte, hex.DecodedLen(len(emptyRDB)))

	_, err := hex.Decode(body, []byte(emptyRDB))
	if err != nil {
		return nil, fmt.Errorf("PSYNC: failed to decode RDB file: %s\n", err.Error())
	}
	return []byte(fmt.Sprintf("+FULLRESYNC %s %s\r\n$%d\r\n%s", replid, repl_offset, len(body), body)), nil
}

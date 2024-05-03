package commands

import (
	"fmt"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/instance"
)

type WaitCommand struct {
	NumReplicas int
	Timeout     int
}

func (cmd *WaitCommand) Execute(inst *instance.Instance) ([]byte, error) {
	fmt.Printf("Wait command with offset %d\n", inst.Offset)
	if inst.Offset == 0 {
		fmt.Println("Master has not propagated any commands")
		return []byte(fmt.Sprintf(":%d\r\n", inst.NumReplicas())), nil
	}

	endTime := time.Now().Add(time.Duration(cmd.Timeout) * time.Millisecond)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()

	inst.SetAckCnt(0)
	inst.SendReplAck()

	for {
		select {
		case <-inst.AckChan:
			if inst.GetAckCnt() >= cmd.NumReplicas {
				return []byte(fmt.Sprintf(":%d\r\n", inst.GetAckCnt())), nil
			}
		case <-tick.C:
			if time.Now().After(endTime) {
				return []byte(fmt.Sprintf(":%d\r\n", inst.GetAckCnt())), nil
			}
		}
	}

	return []byte(fmt.Sprintf(":%d\r\n", inst.GetAckCnt())), nil
}

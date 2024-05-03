package instance

import (
	"fmt"
	"net"
	"sync"
)

type Instance struct {
	Store     Store
	Info      map[string]map[string]string
	Replicas  []net.Conn
	ReplMutex sync.RWMutex
	Master    net.Conn
	Offset    int

	ackMtx  sync.RWMutex
	numAck  int
	AckChan chan struct{}
}

func (inst *Instance) NumReplicas() int {
	inst.ReplMutex.Lock()
	n := len(inst.Replicas)
	inst.ReplMutex.Unlock()

	return n
}

func (inst *Instance) GetReplicas() []net.Conn {
	inst.ReplMutex.Lock()
	conns := make([]net.Conn, len(inst.Replicas))
	copy(conns, inst.Replicas)
	inst.ReplMutex.Unlock()

	return conns
}

func (inst *Instance) AddReplica(conn net.Conn) {
	inst.ReplMutex.Lock()
	inst.Replicas = append(inst.Replicas, conn)
	inst.ReplMutex.Unlock()
}

func (inst *Instance) IncrementACK() {
	inst.ackMtx.Lock()
	inst.numAck++
	inst.ackMtx.Unlock()
	inst.AckChan <- struct{}{}
}

func (inst *Instance) GetAckCnt() int {
	inst.ackMtx.Lock()
	defer inst.ackMtx.Unlock()
	return inst.numAck
}

func (inst *Instance) SetAckCnt(cnt int) {
	inst.ackMtx.Lock()
	defer inst.ackMtx.Unlock()
	inst.numAck = cnt
}

func (inst *Instance) SendReplAck() {
	repls := inst.GetReplicas()
	for _, conn := range repls {
		msg := []byte("*3\r\n$8\r\nreplconf\r\n$6\r\nGETACK\r\n$1\r\n*\r\n")
		_, err := conn.Write(msg)

		if err != nil {
			fmt.Printf("Error sending REPLCONF GETACK * to replicas\n")
		}
	}
}

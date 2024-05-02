package instance

import (
	"net"
	"sync"
)

type Instance struct {
	Store     Store
	Info      map[string]map[string]string
	Replicas  []net.Conn
	ReplMutex sync.RWMutex
	Master    net.Conn
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

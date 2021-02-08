// +build linux

package websockets

import (
	"net"
	"reflect"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

const INIT_FLUSH_COUNT = 1
const MAX_FLUSH_ATTEMPT = 3

type connectionManager struct {
	fd        int
	connLimit int

	connMutex sync.RWMutex
	// connections[FD] => net.Conn
	connections map[int]net.Conn

	// TODO(kano): let's consider moving the connection attempts out of this data structure and
	//     adding it in the connections.
	assetMutex sync.RWMutex
	// assetFDs[BTC.BTC] => map[FD] => connection attempts
	assetFDs map[string]map[int]int
}

func ConnectionManagerInit(connLimit int) (*connectionManager, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		logger.Warnf("mkepoll err %v", err)
		return nil, err
	}
	return &connectionManager{
		fd:          fd,
		connections: make(map[int]net.Conn),
		assetFDs:    make(map[string]map[int]int),
		connLimit:   connLimit,
	}, nil
}

func (cm *connectionManager) GetConnection(fd int) *net.Conn {
	cm.connMutex.RLock()
	defer cm.connMutex.RUnlock()
	ret, ok := cm.connections[fd]
	if !ok {
		return nil
	}
	return &ret
}

func (cm *connectionManager) Add(conn net.Conn) error {
	// Extract file descriptor associated with the connection
	fd := websocketFD(conn)
	err := unix.EpollCtl(cm.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		logger.Warnf("add epoll fail %v", err)
		return err
	}
	cm.connMutex.Lock()
	defer cm.connMutex.Unlock()
	cm.connections[fd] = conn
	//TODO(acsaba): add some metric for len(e.connections)
	return nil
}

func (cm *connectionManager) Remove(conn net.Conn) {
	fd := websocketFD(conn)
	err := unix.EpollCtl(cm.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		logger.Warnf("epoll remove error %v", err)
		return
	}
	cm.connMutex.Lock()
	defer cm.connMutex.Unlock()
	delete(cm.connections, fd)
	// TODO(kano): Not sure if closing has to wait for a client turnaround.
	//     Consider close before locking, if it's safe to do so.
	conn.Close()
	//TODO(acsaba): add some metric for len(e.connections)
}

// TODO(kano): document if this only works for existing connections, or it also accepts new ones.
func (cm *connectionManager) WaitOnReceive() (map[int]net.Conn, error) {
	const maxEventNum = 100
	const waitMSec = 100

	events := make([]unix.EpollEvent, maxEventNum)
	n, err := unix.EpollWait(cm.fd, events, waitMSec)
	if err != nil {
		if err.Error() != "interrupted system call" {
			logger.Warnf("failed on wait %v", err)
		}
		return nil, err
	}
	cm.connMutex.RLock()
	defer cm.connMutex.RUnlock()
	readableConnections := map[int]net.Conn{}
	for i := 0; i < n; i++ {
		conn, found := cm.connections[int(events[i].Fd)]
		if found {
			readableConnections[int(events[i].Fd)] = conn
		}
		// TODO(kano): what handle or document !found.
	}
	return readableConnections, nil
}

func websocketFD(conn net.Conn) int {
	// TODO(kano): discuss why we need reflection, or if we can go around it.
	// This is how i found how to do it, once all is working well we can work on optimising it
	// for now though the whole pattern does hinge on getting an FD of the conn,
	// It's only called once onConnect, I would rather not optimise it until everything else is good.
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

	return int(pfdVal.FieldByName("Sysfd").Int())
}

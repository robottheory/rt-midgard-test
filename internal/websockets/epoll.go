package websockets

import (
	"net"
	"reflect"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// TODO(kano): Merge this with pools map in websockets, or rename.
type epoll struct {
	sync.RWMutex

	fd          int
	connections map[int]net.Conn
}

func MkEpoll() (*epoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		logger.Warnf("mkepoll err %v", err)
		return nil, err
	}
	return &epoll{
		fd:          fd,
		connections: make(map[int]net.Conn),
	}, nil
}

func (e *epoll) Add(conn net.Conn) error {
	// Extract file descriptor associated with the connection
	fd := websocketFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		logger.Warnf("add epoll fail %v", err)
		return err
	}
	e.Lock()
	defer e.Unlock()
	e.connections[fd] = conn
	//TODO(acsaba): add some metric for len(e.connections)
	return nil
}

func (e *epoll) Remove(conn net.Conn) error {
	fd := websocketFD(conn)
	err := unix.EpollCtl(e.fd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		logger.Warnf("epoll remove error %v", err)
		return err
	}
	e.Lock()
	defer e.Unlock()
	delete(e.connections, fd)
	//TODO(acsaba): add some metric for len(e.connections)
	return nil
}

func (e *epoll) Wait() ([]net.Conn, error) {
	const maxEventNum = 100
	const waitMSec = 100

	events := make([]unix.EpollEvent, maxEventNum)
	n, err := unix.EpollWait(e.fd, events, waitMSec)
	if err != nil {
		logger.Warnf("failed on wait %v", err)
		return nil, err
	}
	// TODO(kano): use Lock because e.connections is modified by the getter when key is missing.
	e.RLock()
	defer e.RUnlock()
	var connections []net.Conn
	for i := 0; i < n; i++ {
		// TODO(kano): Let's discuss. Is the FD key here the same as in epoll.Add?
		//    We add it to the map here and in Add too, right?
		//    Let's add some documentation about it after the discussion.
		conn := e.connections[int(events[i].Fd)]
		connections = append(connections, conn)
	}
	return connections, nil
}

func websocketFD(conn net.Conn) int {
	// TODO(kano): discuss why we need reflection, or if we can go around it.
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

	return int(pfdVal.FieldByName("Sysfd").Int())
}

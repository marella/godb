package zmq

import (
	"net"
	"time"
)

const (
	BUFSIZE = 1024
)

type Socket struct {
	conn     net.Conn
	bufsize  int
	listener net.Listener
}

func NewSocket(bufsize int) (soc *Socket, err error) {
	soc = &Socket{bufsize: bufsize}
	return
}

func (soc *Socket) Listen(port string) (err error) {
	soc.listener, err = net.Listen("tcp", ":"+port)
	return
}

func (soc *Socket) Accept() (err error) {
	soc.conn, err = soc.listener.Accept()
	return
}

func (soc *Socket) Connect(addr string) (err error) {
	conn, err := net.DialTimeout("tcp", addr, 50 * time.Millisecond)
	if isError(err) {
		return
	}
	soc.conn = conn
	return
}

func (soc *Socket) Close() (err error) {
	err = soc.conn.Close()
	return
}

func (soc *Socket) RecvBytes() (data []byte, err error) {
	buf := make([]byte, soc.bufsize)

	n, err := soc.conn.Read(buf[0:])
	if isError(err) {
		return
	}
	size := int(buf[0])
	data = append(data, buf[1:n]...)

	for i := 0; i < size; i++ {
		n, err = soc.conn.Read(buf[0:])
		if isError(err) {
			return
		}
		data = append(data, buf[:n]...)
	}
	return
}

func (soc *Socket) SendBytes(data []byte) (offset int, err error) {
	size := len(data) + 1
	data = append([]byte(string(size/soc.bufsize)), data...)
	offset = 0
	n := 0
	for offset < len(data) {
		n, err = soc.conn.Write(data[offset:])
		if isError(err) {
			return
		}
		offset += n
	}
	return
}

func isError(err error) bool {
	if err != nil {
		return true
	}
	return false
}

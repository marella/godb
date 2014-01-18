package godb

import (
	"fmt"
	"net"
	"os"
)

const GODB_PORT = "2000"

type Godb struct {
	conn net.Conn
	db string
}

func New(ip string, db string) *Godb {
	conn, err := net.Dial("tcp", ip+":"+GODB_PORT)
	checkError(err)
	//fmt.Fprintf(conn, db)
	return &Godb{conn, db}
}

func (g *Godb) Query(s string) (r string, ok bool) {
	r = "OK"
	ok = false
	BigWrite(g.conn, s)
	s, Ok := BigRead(g.conn)
	if !Ok {
		return
	}
	code := s[0:1]
	if code == "1" {
		return
	} else if code == "-" {
		r = "Error: "
		switch s[1:2] {
			case "C":
				r += "Command not found"
			case "A":
				r += "Insufficient Arguments"
			case "K":
				r += "Key not set"
		}
	} else if code == "R" {
		r = s[1:]
	}
	ok = true
	return
}

func BigRead(conn net.Conn) (s string, ok bool) {
	ok = false
	s = ""
	var buf [512]byte
	n, err := conn.Read(buf[0:])
	if err != nil {
		return
	}
	size := int(buf[0])
	s = string(buf[1:n])
	for i := 0; i < size; i++ {
		n, err = conn.Read(buf[0:])
		if err != nil {
			return
		}
		s += string(buf[:n])
	}
	ok = true
	return
}

func BigWrite(conn net.Conn, s string) bool {
	size := len(s) + 1
	s = string(size/512) + s
	i := 0
	for ;i < size/512; i++ {
		conn.Write([]byte(s[i*512 : (i+1)*512]))
	}
	conn.Write([]byte(s[i*512 : i*512 + size%512]))
	return true
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
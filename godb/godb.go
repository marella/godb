package godb

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Godb struct {
	conn net.Conn
	db string
}

func New(ip string, db string) *Godb {
	conn, err := net.Dial("tcp", ip+":2000")
	checkError(err)
	//fmt.Fprintf(conn, db)
	return &Godb{conn, db}
}

func (g *Godb) Query(s string) (r string, ok bool) {
	fmt.Fprintf(g.conn, s) // send to server
	var buf [512]byte
	n, err := g.conn.Read(buf[0:]) // read server response
	checkError(err)
	r = string(buf[:n])
	ok = true
	if r == "OK" {
		return
	} else if strings.HasPrefix(r, "return") {
		r = r[strings.Index(r, ": ")+2:]
		return
	}
	return
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
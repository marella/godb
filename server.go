package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
)

const GODB_PORT = "2000"

var db map[string]string

var mutex = &sync.Mutex{}

var argc = map[string]int {
	"copy": 2,
	"del": 1,
	"get": 1,
	"quit": 0,
	"rename": 2,
	"set": 2,
}

func GoSQL(sql string) (response string, status bool) {
	response = "1"
	status = true

	sql = strings.Trim(sql, "; ")
	
	// find the command word
	cmd := sql
	if i := strings.Index(sql, " "); i > 0 {
		cmd = sql[0:i]
	}
	cmd = strings.ToLower(cmd) // to support upper and lower case commands
	
	// check if command is present and parameters count is matching
	args := strings.Fields(sql)
	if v, ok := argc[cmd]; !ok {
		response = "-C"
		return
	} else if len(args)-1 < v {
		response = "-A"
		return
	}

	// run the query
	mutex.Lock()
	switch cmd {

		case "copy":
			if args[1] == args[2] {
				break
			}
			if v, ok := db[args[1]]; ok {
				db[args[2]] = v
			} else {
				response = "-A"
			}

		case "del":
			delete(db, args[1])

		case "get":
			if v, ok := db[args[1]]; ok {
				response = fmt.Sprint("R", v)
			} else {
				response = "-K"
			}

		case "quit":
			status = false // to break the loop

		case "rename":
			if args[1] == args[2] {
				break
			}
			if v, ok := db[args[1]]; ok {
				db[args[2]] = v
				delete(db, args[1])
			} else {
				response = "-K"
			}

		case "set":
			db[args[1]] = strings.Join(args[2:], " ")

		default:
			response = "-C"

	}
	mutex.Unlock()
	return
}

func LoadDB() {
	buf, err := ioutil.ReadFile("db.txt")
	if err != nil {
		return
	}
	b := bytes.NewBuffer(buf)
	d := gob.NewDecoder(b)

    // Decoding the serialized data
    err = d.Decode(&db)
    if err != nil {
        //panic(err)
        return
    }
}

func SaveDB() {
	b := new(bytes.Buffer)

    e := gob.NewEncoder(b)

    // Encoding the map
    err := e.Encode(db)
    if err != nil {
        //panic(err)
        return
    }
	ioutil.WriteFile("db.txt", b.Bytes(), 0777)
}

func MultiSQL(sql string) (response string, status bool) {
	response = ""
	status = true
	sql = strings.Trim(sql, "; ")
	msql := strings.Split(sql, ";")
	for i := 0; i < len(msql); i++ {
		r, ok := GoSQL(msql[i])
		response += r + "; "
		if !ok {
			status = false
			return
		}
	}
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

func handleClient(conn net.Conn) {

	defer conn.Close()

	//var buf [512]byte
	for {
		sql, ok := BigRead(conn)
		if !ok {
			return
		}
		r, ok := MultiSQL(sql)
		r = strings.Trim(r, "; ")
		BigWrite(conn, r)
		if !ok {
			mutex.Lock()
			SaveDB()
			mutex.Unlock()
			return
		}
	}
}

func main() {
	db = make(map[string]string)

	listener, err := net.Listen("tcp", ":"+GODB_PORT)
	checkError(err)

	LoadDB()
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go handleClient(conn)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
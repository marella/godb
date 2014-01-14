package main

import (
	"fmt"
	"strings"
	"os"
	"net"
	"io/ioutil"
	"encoding/gob"
	"bytes"
)

const GODB_PORT = "2000"

var db map[string]string

var argc = map[string]int {
	"copy": 2,
	"del": 1,
	"get": 1,
	"quit": 0,
	"rename": 2,
	"set": 2,
}

func GoSQL(sql string) (response string, status bool) {
	response = "OK"
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
		response = fmt.Sprintf("error: '%s' command not found", cmd)
		return
	} else if len(args)-1 < v {
		response = fmt.Sprintf("error: '%s' expects %d parameters but only %d given", cmd, v, len(args)-1)
		return
	}

	// run the query
	switch cmd {

		case "copy":
			if v, ok := db[args[1]]; ok {
				db[args[2]] = v
			} else {
				response = fmt.Sprintf("error: '%s' is not set", args[1])
			}

		case "del":
			delete(db, args[1])

		case "get":
			if v, ok := db[args[1]]; ok {
				response = fmt.Sprint("return: ", v)
			} else {
				response = fmt.Sprintf("error: '%s' is not set", args[1])
			}

		case "quit":
			response = "connection closing"
			status = false // to break the loop
			return

		case "rename":
			if v, ok := db[args[1]]; ok {
				db[args[2]] = v
				delete(db, args[1])
			} else {
				response = fmt.Sprintf("error: '%s' is not set", args[1])
			}

		case "set":
			db[args[1]] = strings.Join(args[2:], " ")

		default:
			response = fmt.Sprintf("error: '%s' command not found", cmd)

	}
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
        panic(err)
    }
}

func SaveDB() {
	b := new(bytes.Buffer)

    e := gob.NewEncoder(b)

    // Encoding the map
    err := e.Encode(db)
    if err != nil {
        panic(err)
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
		if strings.HasPrefix(r, "error") {
			return
		}
		if !ok {
			status = false
			return
		}
	}
	return
}

func handleClient(conn net.Conn) {

	defer conn.Close()

	var buf [512]byte
	for {
		n, err := conn.Read(buf[0:])
		if err != nil {
			return
		}
		sql := string(buf[:n])
		r, ok := MultiSQL(sql)
		r = strings.Trim(r, "; ")
		fmt.Fprintf(conn, r)
		if !ok {
			SaveDB()
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
package main

import (
	"fmt"
	"github.com/marella/godb/godb"
	"testing"
	"os/exec"
	"time"
)

func TestMultiSQL(t *testing.T) {

	go func() {
		cmd := exec.Command("go run server.go &")
		cmd.Run()
	}()
	time.Sleep(100 * time.Millisecond)
	sql := "set a hello world" //; copy a b; rename b c; get c; del c; get c;
	g := godb.New("127.0.0.1", "db_name")
	r, ok := g.Query(sql)
	if !ok {
		// 
	}
	fmt.Println(r)
}
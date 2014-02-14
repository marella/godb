package main

import (
	"fmt"
	"github.com/marella/godb/godb"
	"os/exec"
	"sync"
	"testing"
	"time"
)

var w sync.WaitGroup

func TestMain(t *testing.T) {

	// start the server
	cmd := exec.Command("go", "run", "server.go") // for linux use $ ./godb
	err := cmd.Start()
	if err != nil {
		t.Errorf(err.Error())
	}

	fmt.Println("Test begins in 5 seconds...")
	time.Sleep(5000 * time.Millisecond) // wait for firewall permissions
	
	w.Add(1000)
	// 1000 clients
	for i := 0; i < 1000; i++ {
		go func(i int) {
			g := godb.New("127.0.0.1", "db_name")
			// 10 queries
			for j := 0; j < 10; j++ {
				g.Query(fmt.Sprintf("set test[%d][%d] This is an automated test for client %d and query %d", i, j, i, j))
			}
			g.Query("quit") // to save the map in db.txt
			fmt.Print(i, ",") // to know that a client is finished
			w.Done()
		}(i)
	}
	w.Wait()
	fmt.Println("")

	// now check if above queries went well
	g := godb.New("127.0.0.1", "db_name")
	r, ok := g.Query("get test[1][2]")
	if ok {
		fmt.Println("get test[1][2] returned:", r)
	} else {
		fmt.Println("Something went wrong. Test failed.")
	}
}
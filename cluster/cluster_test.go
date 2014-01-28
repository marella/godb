package cluster

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	N         = 100 // Number of servers in cluster - 100, 200, 400, 800
	M         = 1   // Number of messages to broadcast by each server
	TIMEDELAY = 0  // Timedelay in seconds between each broadcast of a server
	TIMEOUT   = 10 // Timeout in seconds for receiving messages
	// View the README.md file to know the best parameters obtained from test results
)

func TestCluster(t *testing.T) {
	N := 1000
	fmt.Println("---------------------------------------------------")
	fmt.Println("Begin Cluster test:", N)

	c, err := New("examples/cluster_test")
	if isError(err) {
		t.Fatal("Error: Could not load cluster:", err.Error())
	}

	fmt.Println("Started adding", N, "servers to cluster...")
	for i := 1; i <= N; i++ {
		addr := "127.0.0.1:" + strconv.Itoa(10000+i)
		c.Add(i, addr)
	}

	fmt.Println("Checking if servers are added successfully...")
	for i := 1; i <= N; i++ {
		addr := "127.0.0.1:" + strconv.Itoa(10000+i)
		if !c.Exists(i) || !c.AddressExists(addr) {
			t.FailNow()
		}
	}

	fmt.Println("Removing", N, "servers from cluster...")
	for i := 1; i <= N; i++ {
		err = c.Remove(i)
		if isError(err) {
			t.FailNow()
		}
	}

	fmt.Println("Checking if servers are removed successfully...")
	for i := 1; i <= N; i++ {
		addr := "127.0.0.1:" + strconv.Itoa(10000+i)
		if c.Exists(i) || c.AddressExists(addr) {
			t.FailNow()
		}
	}

	if t.Failed() {
		fmt.Println("Cluster test FAILED.")
	} else {
		fmt.Println("Cluster test PASSED.")
	}
	fmt.Println("---------------------------------------------------")
}

func TestServer(t *testing.T) {
	fmt.Println("Begin Server test:", N, "x", M)

	c, err := New("examples/cluster_" + strconv.Itoa(N))
	if isError(err) {
		t.Fatal("Error: Could not load cluster:", err.Error())
	}
	n := len(c.peers)

	var start, w sync.WaitGroup
	var done Counter
	w.Add(n)
	start.Add(n)

	fmt.Println("Started creating", n, "servers in cluster...")
	for i := 1; i <= n; i++ {
		go func(i int) {
			defer w.Done()

			s, err := c.New(i)
			if isError(err) {
				t.Fatal("Error: Could not create server:", err.Error())
			}

			msg, err := c.Compose(BROADCAST, i)
			if isError(err) {
				t.Fatal("Error: Could not compose message:", err.Error())
			}

			start.Done()
			// Wait till all servers are started
			start.Wait()

			for j := 0; j < M; j++ {
				time.Sleep(TIMEDELAY * time.Second)
				s.Outbox() <- msg
			}

			for j := 0; j < M*(n-1); j++ {
				if t.Failed() {
					return
				}

				select {
				case e := <-s.Inbox():
					if e.Msg != e.Pid {
						t.FailNow()
					}
				case <-time.After(TIMEOUT * time.Second):
					t.FailNow()
				}
			}

			done.Count()
		}(i)
	}

	fmt.Println("Waiting for all servers to finish messaging...")
	w.Wait()

	if t.Failed() || done.Count() != int64(n+1) {
		fmt.Println("Server test FAILED.")
	} else {
		fmt.Println("Server test PASSED.")
	}
	fmt.Println("---------------------------------------------------")
}

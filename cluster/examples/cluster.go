package main

import (
	"github.com/marella/godb/cluster"

	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	myid := 1

	args := os.Args
	if len(args) > 1 {
		myid, _ = strconv.Atoi(args[1])
	}

	c, _ := cluster.New("cluster")
	c.Add(3, "127.0.0.1:10003")
	c.Update(3, "127.0.0.1:10005")
	c.Remove(3)

	server, _ := c.New(myid)

	fmt.Println("Current cluster configuration:", c)
	fmt.Println("Press Enter to start server", myid, "...")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	msg, _ := c.Compose(cluster.BROADCAST, "hello there")
	server.Outbox() <- msg

	select {
	case envelope := <-server.Inbox():
		fmt.Printf("Received msg from %d: '%s' MsgId: %d\n", envelope.Pid, envelope.Msg, envelope.MsgId)

	case <-time.After(10 * time.Second):
		println("Waited and waited. Ab thak gaya")
	}
}

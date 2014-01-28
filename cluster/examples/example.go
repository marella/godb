package main

import (
	"github.com/marella/godb/cluster"

	"fmt"
)

func main() {
	c, _ := cluster.New("cluster")
	c.Add(3, "127.0.0.1:10003")
	c.Update(3, "127.0.0.1:10005")
	c.Remove(3)

	s1, _ := c.New(1)
	s2, _ := c.New(2)

	fmt.Println("Current cluster configuration:", c)

	msg, _ := c.Compose(cluster.BROADCAST, "hello there")
	s1.Outbox() <- msg
	msg, _ = c.Compose(cluster.BROADCAST, "yeah")
	s2.Outbox() <- msg

	e := <-s2.Inbox()
	fmt.Printf("Received msg from %d: '%s' MsgId: %d\n", e.Pid, e.Msg, e.MsgId)
	e = <-s1.Inbox()
	fmt.Printf("Received msg from %d: '%s' MsgId: %d\n", e.Pid, e.Msg, e.MsgId)
}
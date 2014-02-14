package main

import (
	"github.com/marella/godb/raft"

	"fmt"
	"time"
)

func main() {
	c, _ := raft.NewCluster("raft")
	s1, _ := c.New(1)
	s2, _ := c.New(2)
	s3, _ := c.New(3)
	s4, _ := c.New(4)
	s5, _ := c.New(5)
	s1.SetTerm(0)
	s2.SetTerm(0)
	s3.SetTerm(0)
	s4.SetTerm(0)
	s5.SetTerm(0)

	fmt.Println("Current cluster configuration:", c)

	for {
		if s1.IsLeader() {
			fmt.Println("Leader: s1")
		}
		if s2.IsLeader() {
			fmt.Println("Leader: s2")
		}
		if s3.IsLeader() {
			fmt.Println("Leader: s3")
		}
		if s4.IsLeader() {
			fmt.Println("Leader: s4")
		}
		if s5.IsLeader() {
			fmt.Println("Leader: s5")
		}
		fmt.Println(s1.Term(), s2.Term(), s3.Term(), s4.Term(), s5.Term())
		time.Sleep(2 * time.Second)
	}
}

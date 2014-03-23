package main

import (
	"github.com/marella/godb/raft"

	"fmt"
)

func main() {
	c, _ := raft.NewCluster("raft")
	
	s := []Raft{}
	var temp Raft
	for j := 0; j < 5; j++ {
		temp, _ = c.New(j+1)
		s = append(s, temp)
		s[j].SetTerm(0)
	}

	for j := 0; j < 5; j++ {
		if s[j].isLeader() {
			s[j].Inbox() <- "LOG"
			fmt.Println(<-s[j].Outbox())
		}
	}
}

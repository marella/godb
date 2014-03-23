package raft

import (
	"fmt"
	"testing"
	"time"
)

const (
	WAITTIME = 500
	N        = 1
)

func TestReplication(t *testing.T) {
	fmt.Println("---------------------------------------------------")
	fmt.Println("Begin Replication test:")

	c, err := NewCluster("examples/raft")
	if isError(err) {
		t.Fatal("Error: Could not load cluster:", err.Error())
	}
	s := []Raft{}
	var temp Raft
	for j := 0; j < 5; j++ {
		temp, _ = c.New(j + 1)
		s = append(s, temp)
		s[j].SetTerm(0)
	}

	prevLeader := 0
	fmt.Println("Testing...")
	for i := 0; i < N; i++ {
		time.Sleep(WAITTIME * time.Millisecond)
		count := 0
		leader := 0
		for j := 0; j < len(s); j++ {
			if s[j].isLeader() {
				count++
				leader = j + 1
				fmt.Println("Found leader")
				s[j].Inbox() <- "LOG"
				fmt.Println("Sent log item")
				fmt.Println(<-s[j].Outbox())
			}
		}
		if count == 0 {
			t.Error("No leader elected yet!")
		} else if count > 1 {
			t.Fatal("Too many leaders!", count, "leaders elected")
		}

		if prevLeader != 0 && leader != prevLeader {
			t.Error("Leader changed! prevLeader =", prevLeader, ", leader =", leader)
		}

		prevLeader = leader
	}

	if t.Failed() {
		fmt.Println("Replication test FAILED.")
	} else {
		fmt.Println("Replication test PASSED.")
	}
	fmt.Println("---------------------------------------------------")
}

/*
func TestBasic(t *testing.T) {
	fmt.Println("---------------------------------------------------")
	fmt.Println("Begin Basic test:")

	c, err := NewCluster("examples/raft")
	if isError(err) {
		t.Fatal("Error: Could not load cluster:", err.Error())
	}
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

	prevLeader := 0
	prevTerm := 0
	fmt.Println("Testing...")
	for i := 0; i < N; i++ {
		time.Sleep(WAITTIME * time.Millisecond)
		count := 0
		leader := 0
		if s1.isLeader() {
			// fmt.Println("Leader: s1")
			count++
			leader = 1
		}
		if s2.isLeader() {
			// fmt.Println("Leader: s2")
			count++
			leader = 2
		}
		if s3.isLeader() {
			// fmt.Println("Leader: s3")
			count++
			leader = 3
		}
		if s4.isLeader() {
			// fmt.Println("Leader: s4")
			count++
			leader = 4
		}
		if s5.isLeader() {
			// fmt.Println("Leader: s5")
			count++
			leader = 5
		}

		if count == 0 {
			t.Error("No leader elected yet!")
		} else if count > 1 {
			t.Fatal("Too many leaders!", count, "leaders elected")
		}

		if prevLeader != 0 && leader != prevLeader {
			t.Error("Leader changed! prevLeader =", prevLeader, ", leader =", leader)
		}

		if prevTerm != 0 && s1.Term() != prevTerm {
			t.Error("Term changed! prevTerm =", prevTerm, ", s1.Term() =", s1.Term())
		}
		prevLeader = leader
		prevTerm = s1.Term()
		// fmt.Println(s1.Term(), s2.Term(), s3.Term(), s4.Term(), s5.Term())
	}

	if t.Failed() {
		fmt.Println("Basic test FAILED.")
	} else {
		fmt.Println("Basic test PASSED.")
	}
	fmt.Println("---------------------------------------------------")
}

func TestMinorityFailures(t *testing.T) {
	fmt.Println("---------------------------------------------------")
	fmt.Println("Begin MinorityFailures test:")

	c, err := NewCluster("examples/raft")
	if isError(err) {
		t.Fatal("Error: Could not load cluster:", err.Error())
	}
	s1, _ := c.New(1)
	s2, _ := c.New(2)
	s3, _ := c.New(3)
	s1.SetTerm(0)
	s2.SetTerm(0)
	s3.SetTerm(0)

	prevLeader := 0
	prevTerm := 0
	fmt.Println("Testing...")
	for i := 0; i < N; i++ {
		time.Sleep(WAITTIME * time.Millisecond)
		count := 0
		leader := 0
		if s1.isLeader() {
			// fmt.Println("Leader: s1")
			count++
			leader = 1
		}
		if s2.isLeader() {
			// fmt.Println("Leader: s2")
			count++
			leader = 2
		}
		if s3.isLeader() {
			// fmt.Println("Leader: s3")
			count++
			leader = 3
		}

		if count == 0 {
			t.Error("No leader elected yet!")
		} else if count > 1 {
			t.Fatal("Too many leaders!", count, "leaders elected")
		}

		if prevLeader != 0 && leader != prevLeader {
			t.Error("Leader changed! prevLeader =", prevLeader, ", leader =", leader)
		}

		if prevTerm != 0 && s1.Term() != prevTerm {
			t.Error("Term changed! prevTerm =", prevTerm, ", s1.Term() =", s1.Term())
		}
		prevLeader = leader
		prevTerm = s1.Term()
		// fmt.Println(s1.Term(), s2.Term(), s3.Term())
	}

	if t.Failed() {
		fmt.Println("MinorityFailures test FAILED.")
	} else {
		fmt.Println("MinorityFailures test PASSED.")
	}
	fmt.Println("---------------------------------------------------")
}

func TestMajorityFailures(t *testing.T) {
	fmt.Println("---------------------------------------------------")
	fmt.Println("Begin MajorityFailures test:")

	c, err := NewCluster("examples/raft")
	if isError(err) {
		t.Fatal("Error: Could not load cluster:", err.Error())
	}
	s1, _ := c.New(1)
	s2, _ := c.New(2)
	s1.SetTerm(0)
	s2.SetTerm(0)

	fmt.Println("Testing...")
	for i := 0; i < N; i++ {
		time.Sleep(WAITTIME * time.Millisecond)
		count := 0
		if s1.isLeader() {
			// fmt.Println("Leader: s1")
			count++
		}
		if s2.isLeader() {
			// fmt.Println("Leader: s2")
			count++
		}

		if count == 1 {
			t.Fatal("Leader elected!")
		} else if count > 1 {
			t.Fatal("Too many leaders!", count, "leaders elected")
		}

		// fmt.Println(s1.Term(), s2.Term())
	}

	if t.Failed() {
		fmt.Println("MajorityFailures test FAILED.")
	} else {
		fmt.Println("MajorityFailures test PASSED.")
	}
	fmt.Println("---------------------------------------------------")
}
*/

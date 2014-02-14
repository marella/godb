// Copyright 2014 Ravindra Marella.

// Package raft implements the Raft Leader Election algorithm in Ongaro, Diego, and John Ousterhout. "In search of an understandable consensus algorithm." Draft of October 7 (2013).
// It uses the cluster package to send/receive messages.
//
// Example
//
// First create a cluster c using raft.NewCluster() and then create a server using c.New().
//	package main
//
// 	import (
// 		"github.com/marella/godb/raft"
//
// 		"fmt"
// 		"time"
// 	)
//
// 	func main() {
// 		c, _ := raft.NewCluster("raft")
// 		s1, _ := c.New(1)
// 		s2, _ := c.New(2)
// 		s3, _ := c.New(3)
// 		s4, _ := c.New(4)
// 		s5, _ := c.New(5)
// 		s1.SetTerm(0)
// 		s2.SetTerm(0)
// 		s3.SetTerm(0)
// 		s4.SetTerm(0)
// 		s5.SetTerm(0)
//
// 		fmt.Println("Current cluster configuration:", c)
//
// 		for {
// 			if s1.IsLeader() {
// 				fmt.Println("Leader: s1")
// 			}
// 			if s2.IsLeader() {
// 				fmt.Println("Leader: s2")
// 			}
// 			if s3.IsLeader() {
// 				fmt.Println("Leader: s3")
// 			}
// 			if s4.IsLeader() {
// 				fmt.Println("Leader: s4")
// 			}
// 			if s5.IsLeader() {
// 				fmt.Println("Leader: s5")
// 			}
// 			fmt.Println(s1.Term(), s2.Term(), s3.Term(), s4.Term(), s5.Term())
// 			time.Sleep(2 * time.Second)
// 		}
// 	}
package raft

import (
	"github.com/marella/godb/cluster"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

// Interface to be implemented by a Server in the Cluster.
type Raft interface {
	Term() int
	isLeader() bool

	IsLeader() bool
	SetTerm(int) error
}

// This is an extension of the Cluster in cluster package.
type Cluster struct {
	Terms map[string]string

	HeartBeatRate   time.Duration
	ElectionWaitMin int
	ElectionWaitMax int

	c    *cluster.Cluster
	name string
	ma   sync.Mutex // To protect Addresses map
	mc   sync.Mutex // To make read/write to config file thread safe
}

// Creates and loads the cluster configuration with the given name from <cluster_name>.config and <cluster_name.cluster.config> files in the current directory.
// Configuration is JSON encoded.
func NewCluster(name string) (c *Cluster, err error) {
	if _, err = os.Stat(name + ".config"); isError(err) {
		return
	}
	c = &Cluster{name: name}
	err = c.Load()
	if isError(err) {
		return
	}

	c.c, err = cluster.New(name + ".cluster")
	rand.Seed(time.Now().UTC().UnixNano())
	return
}

// Loads the cluster configuration (Terms and Timeouts) from <cluster_name>.config file in the current directory.
// Configuration is JSON encoded.
func (c *Cluster) Load() (err error) {
	c.mc.Lock()
	defer c.mc.Unlock()
	jsonBlob, err := ioutil.ReadFile(c.name + ".config")
	if isError(err) {
		return
	}

	c.ma.Lock()
	defer c.ma.Unlock()
	err = json.Unmarshal(jsonBlob, &c)
	if isError(err) {
		return
	}
	return
}

func (c *Cluster) save() (err error) {
	b, err := json.MarshalIndent(c, "", "    ")
	if isError(err) {
		return
	}
	return ioutil.WriteFile(c.name+".config", b, 0777)
}

// Saves the current cluster configuration (Terms and Timeouts) to <cluster_name>.config file in the current directory.
// Configuration is JSON encoded.
func (c *Cluster) Save() (err error) {
	c.ma.Lock()
	defer c.ma.Unlock()
	c.mc.Lock()
	defer c.mc.Unlock()
	return c.save()
}

// Creates a new Server that implements the Raft interface.
func (c *Cluster) New(pid int) (r Raft, err error) {
	s := &Server{c: c}
	s.s, err = c.c.New(pid)
	go s.heartBeat()
	go s.listen()
	r = Raft(s)
	return
}

// Get the Term of the server with given pid in the cluster.
func (c *Cluster) Term(pid int) (term int, err error) {
	c.ma.Lock()
	defer c.ma.Unlock()
	t, ok := c.Terms[strconv.Itoa(pid)]
	if !ok {
		err = errors.New("error: server #" + strconv.Itoa(pid) + " does not exist in this cluster!")
		return
	}
	term, _ = strconv.Atoi(t)
	return
}

/* Server */

// Server implements the Raft interface.
type Server struct {
	s        cluster.Server
	c        *Cluster
	isleader bool
	voted    bool
}

// Get the Term of server.
func (s *Server) Term() (term int) {
	term, _ = s.c.Term(s.s.Pid())
	return
}

// Set the Term of server.
func (s *Server) SetTerm(term int) (err error) {
	pid := s.s.Pid()
	c := s.c
	c.ma.Lock()
	defer c.ma.Unlock()
	_, ok := c.Terms[strconv.Itoa(pid)]
	if !ok {
		err = errors.New("error: server #" + strconv.Itoa(pid) + " does not exist in this cluster!")
		return
	}
	c.Terms[strconv.Itoa(pid)] = strconv.Itoa(term)
	c.mc.Lock()
	defer c.mc.Unlock()
	return c.save()
	return
}

func (s *Server) isLeader() bool {
	return s.isleader
}

// Check if the server is a leader.
func (s *Server) IsLeader() bool {
	return s.isLeader()
}

// Leaders send periodic heartbeats.
func (s *Server) heartBeat() {
	for {
		if s.isLeader() {
			msg, _ := s.c.c.Compose(cluster.BROADCAST, 0)
			s.s.Outbox() <- msg
		}
		time.Sleep(s.c.HeartBeatRate * time.Millisecond)
	}
}

// Listen for heartbeats and voterequests.
func (s *Server) listen() {
	for {
		select {
		case msg := <-s.s.Inbox():
			if msg.Msg == 0 {
				term, _ := s.c.Term(msg.Pid)
				if s.isLeader() && term >= s.Term() {
					s.isleader = false
				}
			} else if msg.Msg == 1 {
				s.vote(msg.Pid)
			}

		case <-time.After(time.Duration(s.c.ElectionWaitMin+rand.Intn(s.c.ElectionWaitMax-s.c.ElectionWaitMin)) * time.Millisecond):
			if !s.isLeader() {
				s.voteRequest()
			}
		}
	}
}

// Enter candidate state, begin the election and request for votes.
func (s *Server) voteRequest() {
	s.SetTerm(s.Term() + 1)
	msg, _ := s.c.c.Compose(cluster.BROADCAST, 1)
	s.s.Outbox() <- msg
	s.voted = true
	count := 1
	for {
		if count > len(s.c.Terms)/2 {
			s.isleader = true
			return
		}
		select {
		case msg := <-s.s.Inbox():
			if msg.Msg == "+" {
				count++
			} else if msg.Msg == 0 {
				term, _ := s.c.Term(msg.Pid)
				if term >= s.Term() {
					return
				}
			} else if msg.Msg == 1 {
				term, _ := s.c.Term(msg.Pid)
				if term > s.Term() {
					s.vote(msg.Pid)
					return
				}
			}

		case <-time.After(time.Duration(s.c.ElectionWaitMin+rand.Intn(s.c.ElectionWaitMax-s.c.ElectionWaitMin)) * time.Millisecond):
			s.voteRequest()
			return
		}
	}
}

// Vote for a candidate with the given pid.
func (s *Server) vote(pid int) {
	if s.s.Pid() == pid {
		return
	}

	term, _ := s.c.Term(pid)
	if term > s.Term() {
		s.SetTerm(term)
		s.voted = false
		s.isleader = false
	} else if term < s.Term() {
		return
	}

	if !s.voted {
		msg, _ := s.c.c.Compose(pid, "+")
		s.s.Outbox() <- msg
		s.voted = true
	}
}

/* Others */

func isError(err error) bool {
	if err != nil {
		fmt.Println("Error:", err.Error())
		return true
	}
	return false
}

// Copyright 2014 Ravindra Marella.

// Package raft implements the Raft Leader Election algorithm in Ongaro, Diego, and John Ousterhout. "In search of an understandable consensus algorithm." Draft of October 7 (2013).
// It uses the cluster package to send/receive messages.
//
// Example
//
// First create a cluster c using raft.NewCluster() and then create a server using c.New().
//	package main
//	import (
// 		"github.com/marella/godb/raft"
//
// 		"fmt"
// 	)
//
// 	func main() {
// 		c, _ := raft.NewCluster("raft")
//
// 		s := []Raft{}
// 		var temp Raft
// 		for j := 0; j < 5; j++ {
// 			temp, _ = c.New(j+1)
// 			s = append(s, temp)
// 			s[j].SetTerm(0)
// 		}
//
// 		for j := 0; j < 5; j++ {
// 			if s[j].isLeader() {
// 				s[j].Inbox() <- "LOG"
// 				fmt.Println(<-s[j].Outbox())
// 			}
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
	Leader() int
	SetTerm(int) error
	// Mailbox for state machine layer above to send commands of any
	// kind, and to have them replicated by raft.  If the server is not
	// the leader, an error message with the current leader Pid is returned
	Inbox() chan<- interface{}

	//Mailbox for state machine layer above to receive commands. These
	//are guaranteed to have been replicated on a majority
	Outbox() <-chan interface{}

	//Remove items with index less than given index (inclusive),
	//and reclaim disk space.
	DiscardUpto(index int64)
}

// Identifies an entry in the log
type LogEntry struct {
	// An index into an abstract 2^64 size array
	Index int64

	// The data that was supplied to raft's inbox
	Data interface{}

	Commit bool
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
	s.outbox = make(chan interface{}, 10)
	s.inbox = make(chan interface{}, 10)
	s.s, err = c.c.New(pid)
	if isError(load(&s.Log, strconv.Itoa(s.s.Pid())+".log")) {
		s.Log = make(map[string]LogEntry)
	}
	go s.appendEntry()
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
	leader   int
	outbox   chan interface{}
	inbox    chan interface{}
	// Store the log here
	Log map[string]LogEntry
	ack []int
}

// Mailbox for state machine layer above to send commands of any
// kind, and to have them replicated by raft.  If the server is not
// the leader, an error message with the current leader Pid is returned
func (s *Server) Inbox() chan<- interface{} {
	return s.inbox
}

//Mailbox for state machine layer above to receive commands. These
//are guaranteed to have been replicated on a majority
func (s *Server) Outbox() <-chan interface{} {
	return s.outbox
}

//Remove items with index less than given index (inclusive),
//and reclaim disk space.
func (s *Server) DiscardUpto(index int64) {
	for k, _ := range s.Log {
		i, _ := strconv.ParseInt(k, 10, 64)
		if i > index {
			break
		}
		delete(s.Log, k)
	}
	save(s.Log, strconv.Itoa(s.s.Pid())+".log")
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

// Get the leader Pid.
func (s *Server) Leader() int {
	return s.leader
}

// Create a new log entry with unique index
func newLogEntry(entry interface{}) (logEntry *LogEntry) {
	t := time.Now()
	logEntry = &LogEntry{Index: t.UnixNano(), Data: entry}
	return
}

// Leaders send appendEntries.
func (s *Server) appendEntry() {
	for {
	appendEntryLabel:
		if s.isLeader() {
			s.leader = s.s.Pid()
		}
		select {
		case entry := <-s.inbox:
			if s.isLeader() {
				logItem := newLogEntry(entry)
				logEntry, _ := json.Marshal(logItem)
				msg, _ := s.c.c.Compose(cluster.BROADCAST, logEntry)
				s.ack = []int{}
				s.s.Outbox() <- msg
				for len(s.ack) < len(s.c.Terms)/2 {
					for _, id := range s.s.Peers() {
						if inArray(id, s.ack) {
							msg, _ = s.c.c.Compose(id, 0)
							s.s.Outbox() <- msg
						} else {
							msg, _ = s.c.c.Compose(id, logEntry)
							s.s.Outbox() <- msg
						}
					}
					time.Sleep(s.c.HeartBeatRate * time.Millisecond / 2)
					if !s.isLeader() {
						goto appendEntryLabel
					}
				}

				// Send to commit
				logItem.Commit = true
				logEntry, _ = json.Marshal(logItem)
				msg, _ = s.c.c.Compose(cluster.BROADCAST, logEntry)
				s.s.Outbox() <- msg

				// Wait till committed
				time.Sleep(s.c.HeartBeatRate * time.Millisecond / 2)

				// Send response to client
				s.outbox <- "Appended: " + strconv.FormatInt(logItem.Index, 10)

			} else {
				s.outbox <- fmt.Sprint("Error: Leader =", s.Leader())
			}
		case <-time.After(s.c.HeartBeatRate * time.Millisecond):
			if s.isLeader() {
				msg, _ := s.c.c.Compose(cluster.BROADCAST, 0)
				s.s.Outbox() <- msg
			}
		}
	}
}

// Listen for appendEntries and voterequests.
func (s *Server) listen() {
	for {
		select {
		case msg := <-s.s.Inbox():
			switch msg.Msg.(type) {
			case int:
				if msg.Msg == 0 {
					s.leader = msg.Pid
					term, _ := s.c.Term(msg.Pid)
					if s.isLeader() && term >= s.Term() {
						s.isleader = false
					}
				} else if msg.Msg == 1 {
					s.vote(msg.Pid)
				}
			case string:
				if msg.Msg == "OK" {
					if s.isLeader() && !inArray(msg.Pid, s.ack) {
						s.ack = append(s.ack, msg.Pid)
					}
				}
			case []byte:
				var logEntry LogEntry
				json.Unmarshal(msg.Msg.([]byte), &logEntry)
				s.Log[strconv.FormatInt(logEntry.Index, 10)] = logEntry
				save(s.Log, strconv.Itoa(s.s.Pid())+".log")
				msg, _ = s.c.c.Compose(msg.Pid, "OK")
				s.s.Outbox() <- msg
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

// Save variable to a file
func save(m interface{}, filename string) (err error) {
	b, err := json.MarshalIndent(m, "", "    ")
	if isError(err) {
		return
	}
	return ioutil.WriteFile("log/"+filename, b, 0777)
}

// Load variable from a file
func load(m interface{}, filename string) (err error) {
	jsonBlob, err := ioutil.ReadFile("log/" + filename)
	if isError(err) {
		return
	}
	err = json.Unmarshal(jsonBlob, &m)
	if isError(err) {
		return
	}
	return
}

// Check if a value is in array
func inArray(value int, array []int) bool {
	for _, a := range array {
		if a == value {
			return true
		}
	}
	return false
}

func isError(err error) bool {
	if err != nil {
		fmt.Println("Error:", err.Error())
		return true
	}
	return false
}

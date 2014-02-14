// Copyright 2014 Ravindra Marella.

// Package cluster is a simple clustering library.
// It implements functions to create a cluster and send messages between peers.
// Currently it has no dependencies. A dummy zmq package is created by me in the zmq folder
// to use as a socket library in the cluster package.
//
// Example
//
// First create a cluster using cluster.New() and then create a server using c.New().
// Use the package's c.Compose() method to create Envelopes.
// 	package main
// 	import (
// 		"github.com/marella/godb/cluster"
//
// 		"fmt"
// 	)
//
// 	func main() {
// 		c, _ := cluster.New("cluster")
// 		c.Add(3, "127.0.0.1:10003")
// 		c.Update(3, "127.0.0.1:10005")
// 		c.Remove(3)
//
// 		s1, _ := c.New(1)
// 		s2, _ := c.New(2)
//
// 		fmt.Println("Current cluster configuration:", c)
//
// 		msg, _ := c.Compose(cluster.BROADCAST, "hello there")
// 		s1.Outbox() <- msg
// 		msg, _ = c.Compose(cluster.BROADCAST, "yeah")
// 		s2.Outbox() <- msg
//
// 		e := <-s2.Inbox()
// 		fmt.Printf("Received msg from %d: '%s' MsgId: %d\n", e.Pid, e.Msg, e.MsgId)
// 		e = <-s1.Inbox()
// 		fmt.Printf("Received msg from %d: '%s' MsgId: %d\n", e.Pid, e.Msg, e.MsgId)
// 	}
package cluster

import (
	"encoding/json"
	"errors"
	// "fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	BROADCAST = -1
)

// A Cluster holds the configuration information like list of peer ids in the cluster, address map etc.
// Each cluster created should have a unique name and the settings are loaded from <cluster_name>.config file in the current directory.
// It also holds the MsgCounter that is added to all Envelopes sent in cluster as a MsgId.
type Cluster struct {
	// Map of pids and addresses of peers in the cluster
	Addresses map[string]string

	// Capacity or buffer size of the messaging channels of peers
	ChanCap int

	name       string
	peers      []int
	msgCounter Counter    // Used to assign unique MsgIds to envelopes
	ma         sync.Mutex // To protect Addresses map
	mc         sync.Mutex // To make read/write to config file thread safe
}

// Creates and loads the cluster configuration with the given name from <cluster_name>.config file in the current directory.
// Configuration is JSON encoded.
func New(name string) (c *Cluster, err error) {
	if _, err = os.Stat(name + ".config"); isError(err) {
		return
	}
	c = &Cluster{name: name}
	c.Load()
	return
}

// Loads the cluster configuration from <cluster_name>.config file in the current directory.
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

	c.peers = make([]int, len(c.Addresses))
	i := 0
	for k, _ := range c.Addresses {
		c.peers[i], err = strconv.Atoi(k)
		if isError(err) {
			continue
		}
		i++
	}
	return
}

// Saves the current cluster configuration to <cluster_name>.config file in the current directory.
// Configuration is JSON encoded.
func (c *Cluster) save() (err error) {
	b, err := json.MarshalIndent(c, "", "    ")
	if isError(err) {
		return
	}
	return ioutil.WriteFile(c.name+".config", b, 0777)
}

// Creates a new Peer that implements the Server interface
func (c *Cluster) New(pid int) (s Server, err error) {
	addr, err := c.Address(pid)
	if isError(err) {
		return
	}
	p := &Peer{pid: pid, cluster: c}
	p.port = addr[strings.Index(addr, ":")+1:]
	p.outbox = make(chan *Envelope, c.ChanCap)
	p.inbox = make(chan *Envelope, c.ChanCap)
	go p.recvMessage()
	go p.sendMessage()
	s = Server(p)
	return
}

// Adds a new peer to the cluster configuration.
// Returns error if pid or addr already exist in the cluster.
func (c *Cluster) Add(pid int, addr string) (err error) {
	if c.Exists(pid) {
		return errors.New("error: server #" + strconv.Itoa(pid) + " already exists in this cluster!")
	}
	if c.AddressExists(addr) {
		return errors.New("error: address " + addr + " already exists in this cluster!")
	}
	c.ma.Lock()
	defer c.ma.Unlock()
	c.Addresses[strconv.Itoa(pid)] = addr
	c.peers = append(c.peers, pid)
	c.mc.Lock()
	defer c.mc.Unlock()
	return c.save()
}

// Updates the address of a peer in the cluster configuration.
// Returns error if pid does not exist or addr already exists in the cluster.
func (c *Cluster) Update(pid int, addr string) (err error) {
	if !c.Exists(pid) {
		return errors.New("error: server #" + strconv.Itoa(pid) + " does not exist in this cluster!")
	}
	if c.AddressExists(addr) {
		return errors.New("error: address " + addr + " already exists in this cluster!")
	}
	c.ma.Lock()
	defer c.ma.Unlock()
	c.Addresses[strconv.Itoa(pid)] = addr
	c.mc.Lock()
	defer c.mc.Unlock()
	return c.save()
}

// Removes a peer from the cluster configuration.
func (c *Cluster) Remove(pid int) (err error) {
	if !c.Exists(pid) {
		return
	}
	c.ma.Lock()
	defer c.ma.Unlock()
	delete(c.Addresses, strconv.Itoa(pid))
	i := 0
	for _, v := range c.peers {
		if v == pid {
			break
		}
		i++
	}
	c.peers = append(c.peers[:i], c.peers[i+1:]...)
	c.mc.Lock()
	defer c.mc.Unlock()
	return c.save()
}

// Checks if a given pid exists in the cluster.
func (c *Cluster) Exists(pid int) (ok bool) {
	c.ma.Lock()
	defer c.ma.Unlock()
	_, ok = c.Addresses[strconv.Itoa(pid)]
	return
}

// Checks if a given addr exists in the cluster.
func (c *Cluster) AddressExists(addr string) bool {
	c.ma.Lock()
	defer c.ma.Unlock()
	for _, v := range c.Addresses {
		if v == addr {
			return true
		}
	}
	return false
}

// Returns the address of a given pid. Returns error if pid does not exist in the cluster.
func (c *Cluster) Address(pid int) (addr string, err error) {
	c.ma.Lock()
	defer c.ma.Unlock()
	addr, ok := c.Addresses[strconv.Itoa(pid)]
	if !ok {
		err = errors.New("error: server #" + strconv.Itoa(pid) + " does not exist in this cluster!")
		return
	}
	return
}

// Composes a message into Envelope.
// Returns error if the given pid is not in the cluster.
func (c *Cluster) Compose(pid int, msg interface{}) (e *Envelope, err error) {
	if !c.Exists(pid) && pid != BROADCAST {
		err = errors.New("error: server #" + strconv.Itoa(pid) + " does not exist in this cluster!")
		return
	}
	e = &Envelope{Pid: pid, MsgId: c.msgCounter.Count(), Msg: msg}
	return
}

// Thread safe counter to assign unquie MsgIds to Envelopes in a cluster.
type Counter struct {
	mu sync.Mutex
	x  int64
}

// Increments and returns the Counter value.
// To be used as a MsgId of an Envelope.
func (c *Counter) Count() (x int64) {
	c.mu.Lock()
	c.x++
	x = c.x
	c.mu.Unlock()
	return
}

/* Others */

func isError(err error) bool {
	if err != nil {
		// fmt.Println("Error:", err.Error())
		return true
	}
	return false
}

package cluster

import (
	//zmq "github.com/pebbe/zmq4"
	"github.com/marella/godb/cluster/zmq"
)

// Interface to be implemented by a Peer in the Cluster.
type Server interface {
	// Id of this server.
	Pid() int

	// List of Pids of other peers in the same cluster.
	Peers() []int

	// Channel to send messages to other peers.
	Outbox() chan *Envelope

	// Channel to receive messages from other peers.
	Inbox() chan *Envelope
}

// Peer implements the Server interface.
type Peer struct {
	cluster *Cluster
	pid     int
	outbox  chan *Envelope
	inbox   chan *Envelope
	port    string
}

// Get the Pid of this server.
func (p *Peer) Pid() int {
	return p.pid
}

// Get list of Pids of other peers in the same cluster.
func (p *Peer) Peers() (peers []int) {
	for _, v := range p.cluster.peers {
		if v != p.Pid() {
			peers = append(peers, v)
		}
	}
	return
}

// Channel to send messages to other peers.
func (p *Peer) Outbox() chan *Envelope {
	return p.outbox
}

// Channel to receive messages from other peers.
func (p *Peer) Inbox() chan *Envelope {
	return p.inbox
}

// Maintains the Inbox channel of the peer.
func (p *Peer) recvMessage() {
	soc, _ := zmq.NewSocket(zmq.BUFSIZE)
	err := soc.Listen(p.port)
	if isError(err) {
		return
	}
	defer soc.Close()

	for {
		err = soc.Accept()
		if isError(err) {
			continue
		}
		b, err := soc.RecvBytes()
		if isError(err) {
			continue
		}
		e, err := OpenEnvelope(b)
		if isError(err) {
			continue
		}
		if !p.cluster.Exists(e.Pid) {
			continue
		}
		p.Inbox() <- e
	}
}

// Handles the Outbox channel of the peer.
func (p *Peer) sendMessage() {
	soc, _ := zmq.NewSocket(zmq.BUFSIZE)
	defer soc.Close()

	for {
		e := <-p.Outbox()
		if !p.cluster.Exists(e.Pid) && e.Pid != BROADCAST {
			continue
		}
		to := []int{e.Pid}
		if e.Pid == BROADCAST {
			to = p.Peers()
		}
		e.Pid = p.Pid()
		for _, id := range to {
			addr, _ := p.cluster.Address(id)
			err := soc.Connect(addr)
			if isError(err) {
				continue
			}
			b, _ := e.Encode()
			soc.SendBytes(b)
			soc.Close()
		}
	}
}

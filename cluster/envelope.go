package cluster

import (
	"bytes"
	"encoding/gob"
)

// An Envelope holds the receiver/sender id, a unique message id and the actual message.
type Envelope struct {
	// On the sender side, Pid identifies the receiving peer.
	// If instead, Pid is set to cluster.BROADCAST, the message is sent to all peers.
	// On the receiver side, the Id is always set to the original sender.
	// If the Id is not found, the message is silently dropped.
	Pid int

	// An id that globally and uniquely identifies the message, meant for duplicate detection at
	// higher levels. It is opaque to this package.
	MsgId int64

	// The actual body of the message.
	Msg interface{}
}

// Encodes an Envelope to bytes.
// Used in sendMessage() of peer.
func (e *Envelope) Encode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(e)
	if isError(err) {
		return nil, err
	}
	return w.Bytes(), nil
}

// Decodes byte array to Envelope.
// Used in recvMessage() of peer.
func OpenEnvelope(buf []byte) (e *Envelope, err error) {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	err = decoder.Decode(&e)
	if isError(err) {
		return
	}
	return
}

package p2p

import "net"

// Peer is an interface representing a remote node in the network.
type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

// Transport is anything that defines communication between peers.
// This could be a TCP connection, a WebRTC connection, or something else.
type Transport interface {
	Addr() string
	Dial(addr string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}

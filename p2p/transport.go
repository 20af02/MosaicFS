package p2p

// Peer is an interface representing a remote node in the network.
type Peer interface {
	Close() error
}

// Transport is anything that defines communication between peers.
// This could be a TCP connection, a WebRTC connection, or something else.
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}

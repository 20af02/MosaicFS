package p2p

import (
	"log"
	"net"
)

// TCPPeer is a remote node over an established TCP connection.
type TCPPeer struct {
	// conn is the underlying TCP connection of the peer.
	conn net.Conn

	// true if we dial and retrieve a conn
	// false if we accept and retrieve a conn
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

// Close implements the Peer interface.
func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport { //Transport{
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}
}

// Consume implements the Transport interface, returning a read-only channel of incoming RPC messages.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
		return err
	}

	// Start accept group
	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			log.Printf("TCP accept error: Failed to accept: %v", err)
			continue
		}

		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		log.Printf("dropping peer connection: %s", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, true)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}
	log.Printf("Accepted connection from %v", peer)

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}
	lenDecodeError := 0
	//Read Loop
	rpc := RPC{}
	for {
		err := t.Decoder.Decode(conn, &rpc)
		// if err == net.ErrClosed {
		// 	return
		// }
		if err != nil {
			lenDecodeError++
			// log.Printf("TCP decode error: %s\n", err)
			return
		}
		rpc.From = conn.RemoteAddr()
		t.rpcch <- rpc
		log.Printf("Received message from %v: %s", peer, rpc)
	}

}

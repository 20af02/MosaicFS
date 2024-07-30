package p2p

import (
	"errors"
	"log"
	"net"
	"sync"
)

// TCPPeer is a remote node over an established TCP connection.
type TCPPeer struct {
	// conn is the underlying TCP connection of the peer.
	net.Conn

	// true if we dial and retrieve a conn
	// false if we accept and retrieve a conn
	outbound bool

	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// Send implements the Peer interface and sends a message to the remote peer.
func (p *TCPPeer) Send(data []byte) error {
	_, err := p.Conn.Write(data)
	return err
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
		rpcch:            make(chan RPC, 1024),
	}
}

// Addr implements the Transport interface, returning the address of the local node accepting connections.
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// Consume implements the Transport interface, returning a read-only channel of incoming RPC messages.
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface, dialing a remote peer at the given address.
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handleConn(conn, true)
	return nil
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
	log.Printf("[%s] transport listening on port: %s\n", "TCP", t.ListenAddr)
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			log.Printf("[%s] accept error: Failed to accept: %v", "TCP", err)
			continue
		}

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		log.Printf("dropping peer connection: %s", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, outbound)

	if err = t.HandshakeFunc(peer); err != nil {
		return
	}
	// log.Printf("Accepted connection from %+v\n", peer)

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}
	// lenDecodeError := 0
	//Read Loop
	for {
		rpc := RPC{}

		err := t.Decoder.Decode(conn, &rpc)
		if err != nil {
			// lenDecodeError++
			// log.Printf("TCP decode error: %s\n", err)
			return
		}

		rpc.From = conn.RemoteAddr().String()

		if rpc.Stream {
			peer.wg.Add(1)
			log.Printf("[%s] incoming stream...\n", rpc.From)
			peer.wg.Wait()
			log.Printf("[%s] stream done\n", rpc.From)
			continue
		}
		t.rpcch <- rpc

	}

}

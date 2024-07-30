package p2p

const (
	IncommingMessageT byte = 0x1
	IncomingStreamT   byte = 0x2
)

// RPC holds any arbitrary data that can be sent over each transport between two nodes in the network.
type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}

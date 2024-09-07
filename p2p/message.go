package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// RPC 持有任意在网络中两个节点被传输的任意数据
type RPC struct {
	Form    string
	Payload []byte
	Stream  bool
}

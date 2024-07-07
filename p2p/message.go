package p2p

import "net"

// RPC 持有任意在网络中两个节点被传输的任意数据
type RPC struct {
	Form    net.Addr
	Payload []byte
}

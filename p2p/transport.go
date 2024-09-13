package p2p

import "net"

// Peer is an interface that represents the remote node
type Peer interface {
	Send([]byte) error
	CloseStream()
	net.Conn
}

// Transport is a handler to help nodes which in the network to communicate with each other
// form(TCP,UDP,ws)
type Transport interface {
	Addr() string
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}

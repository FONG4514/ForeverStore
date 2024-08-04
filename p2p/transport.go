package p2p

// Peer is an interface that represents the remote node
type Peer interface {
	Close() error
}

// Transport is a handler to help nodes which in the network to communicate with each other
// form(TCP,UDP,ws)
type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
}

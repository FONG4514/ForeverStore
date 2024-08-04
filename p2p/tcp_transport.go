package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
)

// TCPPeer represents the remote node over TCP established connection
type TCPPeer struct {
	// 这个是一个与一个peer的底层的连接
	conn net.Conn
	// outbound 意味着如果我们主动询问并且建立一条连接，outbound(出境)为true
	// 否则如果我们accept一次询问并且建立连接，则为false
	outbound bool
}

type TCPTransportOpts struct {
	ListenAddress  string
	ShakeHandsFunc HandshakerFunc
	Decoder        Decoder
	OnPeer         func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcChan  chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcChan:          make(chan RPC),
	}
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

// Consume implements the transport interface,and return read-only channel
// for reading the incoming messages received form another peer in network
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcChan
}

// Close implements the Transport interface.
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return err
	}
	//这里开一个协程用于监听端口并且处理连接
	go t.startAcceptLoop()

	log.Printf("TCP transport listening on port: %s\n", t.ListenAddress)

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		accept, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}

		if err != nil {
			fmt.Printf("TCP accept error:%s\n", err)
		}
		// use a new goroutine to handle request
		// so multiple connections may be served concurrently
		fmt.Printf("new incoming connection:%+v", accept)
		go t.handlerConn(accept)
	}
}

// Temp是占位符，充当msg
type Temp struct{}

func (t *TCPTransport) handlerConn(conn net.Conn) {
	var err error
	defer func() {
		fmt.Printf("dropping peer connection: %s", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, false)

	if err = t.ShakeHandsFunc(conn); err != nil {
		return
	}
	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}
	// Read Loop
	msg := RPC{}
	for {
		err := t.Decoder.Decode(conn, &msg)
		if err != nil {
			fmt.Printf("tcp read error :%s\n", err)
			return
		}
		msg.Form = conn.RemoteAddr()
		t.rpcChan <- msg
	}
}

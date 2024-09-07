package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer represents the remote node over TCP established connection
type TCPPeer struct {
	// The underlying connection of the peer. Which in this case
	// is a TCP connection
	net.Conn
	// outbound 意味着如果我们主动询问并且建立一条连接，outbound(出境)为true
	// 否则如果我们accept一次询问并且建立连接，则为false
	outbound bool
	Wg       *sync.WaitGroup
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
		Conn:     conn,
		outbound: outbound,
		Wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

/*func (t *TCPTransport) ListenAddr() string {
	return t.ListenAddress
}*/

// Consume implements the transport interface,and return read-only channel
// for reading the incoming messages received form another peer in network
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcChan
}

// Close implements the Transport interface.
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Dial implements the Transport interface.
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handlerConn(conn, true)
	return nil
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
		go t.handlerConn(accept, false)
	}
}

// Temp 是占位符，充当msg
type Temp struct{}

func (t *TCPTransport) handlerConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		fmt.Printf("dropping peer connection: %s", err)
		conn.Close()
	}()
	peer := NewTCPPeer(conn, outbound)

	if err = t.ShakeHandsFunc(peer); err != nil {
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
		//在没有广播但是又使用p2p网络的情况下，这个连接会空着，一直在等待流
		err := t.Decoder.Decode(conn, &msg)
		if err != nil {
			fmt.Printf("tcp read error :%s\n", err)
			return
		}

		msg.Form = conn.RemoteAddr().String()

		if msg.Stream {
			peer.Wg.Add(1)
			fmt.Printf("[%s] incoming stream,waiting till stream is done\n", conn.RemoteAddr())
			peer.Wg.Wait()
			fmt.Printf("[%s] stream close,stream done continuing normal read loop\n", conn.RemoteAddr())
			continue
		}

		t.rpcChan <- msg
		//fmt.Printf("msg:%+v inside\n", msg)
	}
}

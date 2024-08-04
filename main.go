package main

import (
	"File_System/p2p"
	"log"
)

func main() {
	/*tcpOpts := p2p.TCPTransportOpts{
		ListenAddress:  ":3000",
		ShakeHandsFunc: p2p.NOPHandshakerFunc,
		Decoder:        p2p.DefaultDecoder{},
		//TODO: Add a onPeer func
	}
	tran := p2p.NewTCPTransport(tcpOpts)
	fileServerOpts := FileServerOpts{
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tran,
		BootstrapNodes:    []string{":4000"},
	}
	s := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(time.Second * 3)
		fmt.Println("sleep over")
		s.Stop()
	}()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}*/
	s1 := makeServer(":3000")
	s2 := makeServer(":4000", ":3000")
	go func() {
		log.Fatal(s1.Start())
	}()
	s2.Start()
}

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress:  listenAddr,
		ShakeHandsFunc: p2p.NOPHandshakerFunc,
		Decoder:        p2p.DefaultDecoder{},
		//TODO: Add a onPeer func
	}
	tran := p2p.NewTCPTransport(tcpOpts)
	fileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tran,
		BootstrapNodes:    nodes,
	}
	s := NewFileServer(fileServerOpts)
	return s
}

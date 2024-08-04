package main

import (
	"File_System/p2p"
	"fmt"
	"log"
	"time"
)

func main() {
	tcpOpts := p2p.TCPTransportOpts{
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
	}
	s := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(time.Second * 3)
		fmt.Println("sleep over")
		s.Stop()
	}()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

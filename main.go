package main

import (
	"File_System/p2p"
	"fmt"
	"log"
)

func main() {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress:  ":4000",
		ShakeHandsFunc: p2p.NOPHandshakerFunc,
		Decoder:        p2p.DefaultDecoder{},
		OnPeer:         OPD,
	}
	tran := p2p.NewTCPTransport(tcpOpts)

	go func() {
		for {
			msg := <-tran.Consume()
			fmt.Printf("msg:%v\n", msg)
		}
	}()
	if err := tran.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}
	select {}
}

func OPD(p p2p.Peer) error {
	p.Close()
	fmt.Println("doing some logic with peer")
	return nil
}

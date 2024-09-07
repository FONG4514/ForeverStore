package main

import (
	"File_System/p2p"
	"bytes"
	"log"
	"time"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddress:  listenAddr,
		ShakeHandsFunc: p2p.NOPHandshakerFunc,
		Decoder:        p2p.DefaultDecoder{},
		//TODO: Add a onPeer func
	}
	tran := p2p.NewTCPTransport(tcpOpts)
	fileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr[1:len(listenAddr)] + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tran,
		BootstrapNodes:    nodes,
	}
	s := NewFileServer(fileServerOpts)
	tran.OnPeer = s.OnPeer
	return s
}

func main() {
	s1 := makeServer(":3000")
	s2 := makeServer(":4000", ":3000")
	go func() {
		log.Fatal(s1.Start())
	}()
	time.Sleep(2 * time.Second)
	go s2.Start()
	time.Sleep(2 * time.Second)
	data := bytes.NewReader([]byte("my big data file here!"))
	s2.Store("myPrivateData", data)

	/*r, err := s2.Get("myPrivateData")
	if err != nil {
		log.Fatal(err)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))*/

	select {}
}

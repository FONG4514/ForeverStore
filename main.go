package main

import (
	"File_System/p2p"
	"bytes"
	"fmt"
	"io/ioutil"
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
		EnKey:             newEncryptKey(),
	}
	s := NewFileServer(fileServerOpts)
	tran.OnPeer = s.OnPeer
	return s
}

func main() {
	s1 := makeServer(":3000")
	s2 := makeServer(":7000", ":3000")
	s3 := makeServer(":5000", ":3000", ":7000")
	go func() {
		log.Fatal(s1.Start())
	}()
	go func() {
		log.Fatal(s2.Start())
	}()
	time.Sleep(2 * time.Second)
	go s3.Start()
	time.Sleep(2 * time.Second)

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("picture_%d.png", i)
		data := bytes.NewReader([]byte("my big data file here!"))
		s3.Store(key, data)

		if err := s3.store.Delete(s3.ID, key); err != nil {
			log.Fatal(err)
		}

		r, err := s3.Get(key)
		if err != nil {
			log.Fatal(err)
		}
		b, err := ioutil.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(b))
	}

}

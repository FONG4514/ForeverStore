package main

import (
	"File_System/p2p"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
)

type FileServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer
	store    *Store
	quitch   chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	return &FileServer{
		store: NewStore(StoreOpts{
			Root:              opts.StorageRoot,
			PathTransformFunc: opts.PathTransformFunc,
		}),
		FileServerOpts: opts,
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

type Payload struct {
	Key  string
	Data []byte
}

func (s *FileServer) broadcast(p *Payload) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	// 由于GOB写入流神奇的将p分两次写入mw，所以这里先一次性写入buf，再交给mw进行写入
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(p); err != nil {
		return err
	}
	n, err := mw.Write(buf.Bytes())
	if err != nil {
		return err
	}
	fmt.Printf("hahahahah print %v bytes to peers", n)
	return nil
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	//1. Store this file to disk
	//2. broadcast this file to all known peers in the network
	fmt.Println("call it StoreData")
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	if err := s.store.Write(key, tee); err != nil {
		log.Fatal(err)
		return err
	}

	p := &Payload{
		Key:  key,
		Data: buf.Bytes(),
	}

	return s.broadcast(p)
}

func (s *FileServer) Stop() {
	close(s.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p

	log.Printf("connected with remote %s", p.RemoteAddr().String())

	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Printf("received msg:%v\n", msg.Payload)
			var p Payload
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&p); err != nil {
				fmt.Printf("this is p:%+v\n", err)
				log.Fatal(err)

			}
			fmt.Printf("on loop : %+v\n", p)
		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			fmt.Println("attempting to connect with remote: ", addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}
	return nil
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.bootstrapNetwork()
	s.loop()
	return nil
}

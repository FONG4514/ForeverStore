package main

import (
	"File_System/p2p"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
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

type Message struct {
	From    string
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

/*type DataMessage struct {
	Key  string
	Data []byte
}*/

func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	// 由于GOB写入流神奇的将p分两次写入mw，所以这里先一次性写入buf，再交给mw进行写入
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	_, err := mw.Write(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

type MessageGetFile struct {
	Key string
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		return s.store.Read(key)
	}
	fmt.Printf("dont have file (%s) locally,fetching from network...", key)

	msg := Message{Payload: MessageGetFile{Key: key}}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}
	for _, peer := range s.peers {
		fmt.Println("receving stream from peer: ", peer.RemoteAddr())

		fileBuffer := new(bytes.Buffer)
		n, err := io.CopyN(fileBuffer, peer, 22)
		if err != nil {
			return nil, err
		}

		fmt.Println("received bytes over the network: ", n)
		fmt.Println(fileBuffer.String())
	}

	select {}

	return nil, nil
}

func (s *FileServer) Store(key string, r io.Reader) error {
	//1. Store this file to disk
	//2. broadcast this file to all known peers in the network
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)
	size, err := s.store.Write(key, tee)

	if err != nil {
		return err
	}
	msg := Message{
		Payload: MessageStoreFile{Key: key, Size: size},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Second * 3)

	//TODO use a multiWriter here
	for _, peer := range s.peers {
		peer.Send([]byte{p2p.IncomingStream})
		written, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}
		fmt.Println("received and written bytes to disk : ", written)
	}

	return nil

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
		log.Println("file server stopped due to error or user quit action")
		s.Transport.Close()
	}()

	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error: ", err)
			}
			if err := s.handleMessage(rpc.Form, &msg); err != nil {
				log.Println("handle message error: ", err)
			}
		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return s.handlerMessageStoreFile(from, v)
	case MessageGetFile:
		return s.handlerMessageGetFile(from, v)
	}
	return nil
}

func (s *FileServer) handlerMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.Key) {
		return fmt.Errorf("cannot read file (%s) from disk,maybe it not exist on disk", msg.Key)
	}
	fmt.Printf("serving file (%s)  over the network\n", msg.Key)
	r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("written %d bytes over the network to %s\n", n, from)

	return nil
}

func (s *FileServer) handlerMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peers", from)
	}
	n, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("writtem %v bytes to disk by handlerMessageStoreFile\n", n)

	peer.(*p2p.TCPPeer).Wg.Done()
	return nil
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

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}

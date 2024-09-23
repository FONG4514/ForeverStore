package main

import (
	"File_System/p2p"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

type FileServerOpts struct {
	// ID of the owner of the storage, which will be used to store all files at that location
	// so we can sync all the files if needed.
	ID                string
	EnKey             []byte
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
	if len(opts.ID) == 0 {
		opts.ID = generateID()
	}
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
	ID   string
	Key  string
	Size int64
}

type MessageGetFile struct {
	ID  string
	Key string
}

/*type DataMessage struct {
	Key  string
	Data []byte
}*/

//func (s *FileServer) stream(msg *Message) error {
//	peers := []io.Writer{}
//	for _, peer := range s.peers {
//		peers = append(peers, peer)
//	}
//	mw := io.MultiWriter(peers...)
//	// 由于GOB写入流神奇的将p分两次写入mw，所以这里先一次性写入buf，再交给mw进行写入
//	buf := new(bytes.Buffer)
//	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
//		return err
//	}
//	_, err := mw.Write(buf.Bytes())
//	if err != nil {
//		return err
//	}
//	return nil
//}

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

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(s.ID, key) {
		fmt.Printf("[%s] serving file (%s) from local disk\n", s.Transport.Addr(), key)
		_, reader, err := s.store.Read(s.ID, key)
		return reader, err
	}
	fmt.Printf("[%s],dont have file (%s) locally,fetching from network...\n", s.Transport.Addr(), key)

	msg := Message{Payload: MessageGetFile{Key: hashKey(key), ID: s.ID}}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 500)

	for _, peer := range s.peers {
		//First read the file size so we can limit the amount of bytes that we read
		//form the connection,so it will not keeping hanging
		var fileSizes int64
		binary.Read(peer, binary.LittleEndian, &fileSizes)

		n, err := s.store.WriteDecrypt(s.EnKey, s.ID, key, io.LimitReader(peer, fileSizes))

		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes over the network from (%s)\n", s.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, reader, err := s.store.Read(s.ID, key)
	return reader, err
}

func (s *FileServer) Store(key string, r io.Reader) error {
	//1. Store this file to disk
	//2. broadcast this file to all known peers in the network
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)
	size, err := s.store.Write(s.ID, key, tee)

	if err != nil {
		return err
	}
	msg := Message{
		Payload: MessageStoreFile{Key: hashKey(key), Size: size + 16, ID: s.ID},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(time.Millisecond * 5)

	peers := []io.Writer{}

	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStream})
	n, err := copyEncrypt(s.EnKey, fileBuffer, mw)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] received and written %d bytes to disk\n", s.Transport.Addr(), n)
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
	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] cannot read file (%s) from disk,maybe it not exist on disk", s.Transport.Addr(), msg.Key)
	}
	fmt.Printf("[%s] serving file (%s)  over the network\n", s.Transport.Addr(), msg.Key)
	fileSize, r, err := s.store.Read(msg.ID, msg.Key)
	if err != nil {
		return err
	}

	if closer, ok := r.(io.ReadCloser); ok {
		fmt.Println("Closing closeStream")
		defer closer.Close()
	}

	peer, ok2 := s.peers[from]
	if !ok2 {
		return fmt.Errorf("peer %s not in map", from)
	}

	//First send the incomingStream byte to the peer, and then we can send
	//the file size as int64

	peer.Send([]byte{p2p.IncomingStream})
	binary.Write(peer, binary.LittleEndian, fileSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] written (%d) bytes over the network to %s\n", s.Transport.Addr(), n, from)

	return nil
}

func (s *FileServer) handlerMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in the peers", from)
	}
	n, err := s.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	fmt.Printf("[%s] writtem %v bytes to disk by handlerMessageStoreFile\n", s.Transport.Addr(), n)

	peer.CloseStream()
	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	for _, addr := range s.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			fmt.Printf("[%s] attempting to connect with remote %s\n", s.Transport.Addr(), addr)
			if err := s.Transport.Dial(addr); err != nil {
				log.Printf("[%s] dial error: %s\n", s.Transport.Addr(), err.Error())
			}
		}(addr)
	}
	return nil
}

func (s *FileServer) Start() error {
	fmt.Printf("[%s] starting fileserver ... \n", s.Transport.Addr())
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

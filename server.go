package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"io"
	"log"
	"sync"

	"github.com/20af02/MosaicFS/crypto"
	"github.com/20af02/MosaicFS/p2p"
)

type FileServerOpts struct {
	ID                string
	EncKey            []byte
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootStrapNodes    []string
	// DB to store our known files
	DBFile string
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
		opts.ID = crypto.GenerateID()
	}

	// ensure db file path exists
	if _, err := os.Stat(opts.DBFile); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(opts.DBFile), 0755); err != nil {
			log.Fatalf("Failed to create db file path: %v", err)
		}
	}
	dbHandle, err := NewDBHandler(opts.ID, opts.DBFile)
	if err != nil {
		log.Fatalf("Failed to create db handler: %v", err)
	}

	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
		dbHandler:         dbHandle,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		// TODO: add peers via channel
		peers: make(map[string]p2p.Peer),
	}
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(&msg); err != nil {
		return err
	}
	for _, peer := range s.peers {
		if err := peer.Send([]byte{p2p.IncommingMessageT}); err != nil {
			return err
		}
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

type Message struct {
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

type MessageDeleteFile struct {
	ID  string
	Key string
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(s.ID, key) {
		log.Printf("[%s] serving file [%s] localy\n", s.Transport.Addr(), key)
		_, r, err := s.store.Read(s.ID, key)

		return r, err
	}

	log.Printf("[%s] don't have file [%s] localy, fetching from network...\n", s.Transport.Addr(), key)

	msg := Message{
		Payload: MessageGetFile{
			ID:  s.ID,
			Key: crypto.HashKey(key),
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(500 * time.Millisecond)
	// Open stream
	for _, peer := range s.peers {
		// n, err := s.store.writeStream(key, peer)
		// First read the filesize to limit amount of bytes read
		// to avoid hanging on the stream
		var fileSize int64
		binary.Read(peer, binary.LittleEndian, &fileSize)
		// n, err := s.store.Write(key, io.LimitReader(peer, fileSize))
		n, err := s.store.WriteDecrypt(s.EncKey, s.ID, key, io.LimitReader(peer, fileSize))
		if err != nil {
			log.Printf("[%s] failed to write to disk: %v", s.Transport.Addr(), err)
			return nil, err
		}
		log.Printf("[%s] received  ([%d]) bytes from (%s)", s.Transport.Addr(), n, peer.RemoteAddr())
		peer.CloseStream()
	}
	_, r, err := s.store.Read(s.ID, key)
	if err == nil {
		// Update the file metadata
		if err := s.store.dbHandler.AddLocalMetaDataToExistingKey(key, s.Transport.Addr()); err != nil {
			fmt.Printf("Error updating file metadata: %v", err)
		}
	}
	return r, err
}

func (s *FileServer) Store(key string, r io.Reader) error {
	// // 1. Store this file to disk
	// // 2. Broadcast this file to all known peers in the network
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := s.store.Write(s.ID, key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			ID:   s.ID,
			Key:  crypto.HashKey(key),
			Size: size + 16, /* IV size */
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)

	peers := []io.Writer{}

	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2p.IncomingStreamT})
	n, err := crypto.CopyEncrypt(s.EncKey, fileBuffer, mw)
	if err != nil {
		return err
	}

	replicaLocs := []string{}
	replicaLocs = append(replicaLocs, s.Transport.Addr() /* Local replica */)
	for _, peer := range s.peers {
		replicaLocs = append(replicaLocs, peer.RemoteAddr().String())
	}

	fmd := &FileMetadata{
		Key:              key,
		Size:             size + 16, /* IV size */
		Replicas:         len(s.peers) + 1,
		ReplicaLocations: replicaLocs,
	}

	if _, err := s.store.dbHandler.UpdateFile(*fmd); err != nil {
		return err
	}
	log.Printf("[%s] received and written: (%d) bytes\n", s.Transport.Addr(), n)

	return nil
}

func (s *FileServer) Delete(key string) error {
	msg := Message{
		Payload: MessageDeleteFile{
			ID:  s.ID,
			Key: crypto.HashKey(key),
		},
	}
	log.Printf("[%s] broadcasting delete message for file (%s)\n", s.Transport.Addr(), key)
	if err := s.broadcast(&msg); err != nil {
		return err
	}
	if !s.store.Has(s.ID, key) {
		// try deleting the metadata from the db
		s.store.dbHandler.DeleteFileMetadata(key)

		return fmt.Errorf("[%s] needs to delete (%s), but it does not exist on disk", s.Transport.Addr(), key)
	}
	log.Printf("[%s] deleting file locally (%s)\n", s.Transport.Addr(), key)
	if err := s.store.dbHandler.DeleteFileMetadata(key); err != nil {
		return err
	}
	return s.store.Delete(s.ID, key)
}

func (s *FileServer) ListFiles() ([]FileMetadata, error) {
	return s.store.dbHandler.ListFiles()
}

func (s *FileServer) Stop() {
	close(s.quitch)

	s.store.dbHandler.Close()

	// return s.Transport.Close()
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.RemoteAddr().String()] = p

	log.Printf("Connected with remote: %s", p.RemoteAddr())
	// TODO: do db exchange here
	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Printf("Shutting down server from error or user quit action")
		s.Transport.Close()
	}()
	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Printf("Failed to decode payload: %v", err)
				// return
			}

			if err := s.handleMessage(rpc.From, &msg); err != nil {
				log.Printf("Failed to handle message: %v", err)
				// return
			}

		case <-s.quitch:
			return
		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		fmt.Printf("Received data message: %+v\n", v)
		return s.handleMessageStoreFile(from, v)
	case MessageGetFile:
		fmt.Printf("Received get message: %+v\n", v)
		return s.handleMessageGetFile(from, v)
	case MessageDeleteFile:
		fmt.Printf("Received delete message: %+v\n", v)
		return s.handleMessageDeleteFile(from, v)
	}
	return nil
}

func (s *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !s.store.Has(msg.ID, msg.Key) {
		return fmt.Errorf("[%s] needs to serve (%s), but it does not exist on disk", s.Transport.Addr(), msg.Key)
	}

	log.Printf("[%s] Sending file (%s) to peer (%s)\n", s.Transport.Addr(), msg.Key, from)

	fileSize, r, err := s.store.Read(msg.ID, msg.Key)
	if err != nil {
		return err
	}
	if rc, ok := r.(io.ReadCloser); ok {
		defer rc.Close()
		// defer fmt.Println("Closed file")
	}

	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("received message from unknown peer: %s", from)
	}

	// First send the "incomingStream" byte to the peer,
	// then send the filesize (int64) and finally the file itself
	peer.Send([]byte{p2p.IncomingStreamT})
	binary.Write(peer, binary.LittleEndian, fileSize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	log.Printf("[%s] wrote (%d) bytes to peer (%s)\n", s.Transport.Addr(), n, from)
	return nil

}

func (s *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("received message from unknown peer: %s", from)
	}
	n, err := s.store.Write(msg.ID, msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	log.Printf("[%s] written (%d) bytes to disk\n", s.Transport.Addr(), n)

	peer.CloseStream()
	return nil
}

func (s *FileServer) handleMessageDeleteFile(from string, msg MessageDeleteFile) error {
	if !s.store.Has(msg.ID, msg.Key) {
		fmt.Printf("[%s] needs to delete (%s), but it does not exist on disk\n", s.Transport.Addr(), msg.Key)
		return nil
	}
	log.Printf("[%s] deleting file (%s) from disk on request from [%s]\n", s.Transport.Addr(), msg.Key, from)
	return s.store.Delete(msg.ID, msg.Key)
}

func (s *FileServer) bootstrapNetwork() error {

	maxAttempts := 3                // Maximum number of retry attempts
	initialDelay := 2 * time.Second // Initial backoff delay

	for _, addr := range s.BootStrapNodes {
		if len(addr) == 0 {
			continue
		}

		var err error
		delay := initialDelay

		// Retry loop
		for attempt := 0; attempt < maxAttempts; attempt++ {
			fmt.Printf("[%s] dialing to remote %s (attempt %d)\n", s.Transport.Addr(), addr, attempt+1)
			if err = s.Transport.Dial(addr); err == nil {
				break // Successful connection, exit retry loop
			}

			log.Printf("[%s] Failed to dial: %v (retrying in %v)", s.Transport.Addr(), err, delay)
			time.Sleep(delay)
			delay *= 2 // Exponential backoff

			delay += time.Duration(rand.Intn(1000)) * time.Millisecond
		}

		// If the retry loop finishes with an error, it's persistent
		if err != nil {
			log.Printf("[%s] Failed to dial after %d attempts: %v", s.Transport.Addr(), maxAttempts, err)
			return nil
		}
	}
	return nil

}

func (s *FileServer) Start() error {
	fmt.Printf("[%s] starting server...\n", s.Transport.Addr())
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
	gob.Register(MessageDeleteFile{})
}

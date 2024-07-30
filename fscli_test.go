package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/20af02/MosaicFS/crypto"
	"github.com/20af02/MosaicFS/p2p"
)

func MakeTestServer(ListenAddr string, BootstrapNodes []string) *FileServer {
	NodeConfig := &NodeConfig{
		ListenAddr:     ListenAddr,
		BootstrapNodes: BootstrapNodes,
		ServerID:       crypto.GenerateID(),
	}

	tcpTransport := p2p.NewTCPTransport(p2p.TCPTransportOpts{
		ListenAddr:    NodeConfig.ListenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	})

	fs := NewFileServer(FileServerOpts{
		ID:                NodeConfig.ServerID,
		EncKey:            crypto.NewEncryptionKey(),
		StorageRoot:       "test" + NodeConfig.ListenAddr[1:] + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootStrapNodes:    NodeConfig.BootstrapNodes,
		DBFile:            "./.env/.db/test_" + NodeConfig.ListenAddr[1:] + "_network.db",
	})
	tcpTransport.OnPeer = fs.OnPeer

	return fs
}

// Test Cobra CLI
func TestCLI(t *testing.T) {
	fs := MakeTestServer(":3000", []string{})
	go func() { fs.Start() }()
	fileData := []byte("my big data file here!")
	fileName := "test.txt"
	// write file to disk
	os.WriteFile(fileName, fileData, 0644)
	defer os.Remove(fileName)

	rootCmd := NewFileServerCLI(fs)
	rootCmd.Find([]string{"store", fileName}) // Find the "store" command
	b := bytes.NewBufferString("")
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"store", fileName})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("Error executing command: %v", err)
	}

	rootCmd.Find([]string{"get", fileName})

	b = bytes.NewBufferString("")
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"get", fileName})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("Error executing command: %v", err)
	}

	rootCmd.Find([]string{"delete", fileName})

	b = bytes.NewBufferString("")
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"delete", fileName})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Error executing command: %v", err)
	}

	// ls
	rootCmd.Find([]string{"ls"})
	b = bytes.NewBufferString("")
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"ls"})
	err = rootCmd.Execute()
	// if we encounter more than 1 newline, then we have more than 1 file (error)
	if err != nil || bytes.Count(b.Bytes(), []byte("\n")) > 1 {
		t.Errorf("Error executing command: %v", err)
	}
	fs.Stop()
	teardown(t, fs.store)

	// delete db file
	if err := os.Remove(fs.DBFile); err != nil {
		t.Errorf("Failed to delete db file: %v", err)
	}
}

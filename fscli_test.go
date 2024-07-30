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
	// fmt.Printf("[%s] BootstrapNodes: %v\n", NodeConfig.ListenAddr, NodeConfig.BootstrapNodes)

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
	// reader := bytes.NewReader(fileData)
	// fs.Store(fileName, reader)

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

// func TestLocalDelete(t *testing.T) {
// 	fs := MakeTestServer(":3000", []string{})
// 	fs2 := MakeTestServer(":4000", []string{":3000"})
// 	go func() { fs.Start() }()
// 	go func() { fs2.Start() }()
// 	fileData := []byte("my big data file here!")
// 	fileName := "test.txt"
// 	// write file to disk
// 	os.WriteFile(fileName, fileData, 0644)
// 	defer os.Remove(fileName)
// 	defer os.Remove(fs.DBFile)
// 	defer os.Remove(fs2.DBFile)
// 	defer teardown(t, fs.store)
// 	defer teardown(t, fs2.store)
// 	defer fs.Stop()
// 	defer fs2.Stop()
// 	// reader := bytes.NewReader(fileData)
// 	// fs.Store(fileName, reader)

// 	rootCmd := NewFileServerCLI(fs)
// 	rootCmd.Find([]string{"store", fileName}) // Find the "store" command
// 	b := bytes.NewBufferString("")
// 	rootCmd.SetOut(b)
// 	rootCmd.SetArgs([]string{"store", fileName})
// 	if err := rootCmd.Execute(); err != nil {
// 		t.Errorf("Error executing command: %v", err)
// 	}
// 	// delete --local (local is a flag)

// 	rootCmd.Find([]string{"delete", fileName})

// 	b = bytes.NewBufferString("")
// 	rootCmd.SetOut(b)
// 	rootCmd.SetErr(b)
// 	rootCmd.SetArgs([]string{"delete", fileName, "--local"})
// 	rootCmd.Flags().Set("local", "True")
// 	err := rootCmd.Execute()
// 	if err != nil {
// 		t.Errorf("Error executing command: %v", err)
// 	}

// 	// Check number of files in the db
// 	rootCmd.Find([]string{"ls"})
// 	b1 := bytes.NewBufferString("")
// 	rootCmd.SetOut(b1)
// 	rootCmd.SetArgs([]string{"ls"})
// 	err = rootCmd.Execute()
// 	// if we encounter less than 2 newline, the file was deleted everywhere not just locally
// 	if err != nil || bytes.Count(b1.Bytes(), []byte("\n")) < 2 {
// 		fmt.Print(bytes.Count(b1.Bytes(), []byte("\n")))
// 		t.Errorf("Error executing command: %v", err)
// 	}

// 	// Get
// 	rootCmd.Find([]string{"get", fileName})
// 	b = bytes.NewBufferString("")
// 	rootCmd.SetOut(b)
// 	rootCmd.SetArgs([]string{"get", fileName})
// 	err = rootCmd.Execute()
// 	if err != nil {
// 		t.Errorf("Error executing command: %v", err)
// 	}

// 	// check ls
// 	rootCmd.Find([]string{"ls"})
// 	b2 := bytes.NewBufferString("")
// 	rootCmd.SetOut(b2)
// 	rootCmd.SetArgs([]string{"ls"})
// 	err = rootCmd.Execute()
// 	// if ls is the same as before, then the file was deleted everywhere (error)
// 	if err != nil || !bytes.Equal(b1.Bytes(), b2.Bytes()) {
// 		t.Errorf("Should have received file: %v", err)
// 	}

// 	// delete fileName
// 	rootCmd.Find([]string{"delete", fileName})
// 	b = bytes.NewBufferString("")
// 	rootCmd.SetOut(b)
// 	rootCmd.SetArgs([]string{"delete", fileName})
// 	err = rootCmd.Execute()
// 	if err != nil {
// 		t.Errorf("Error executing command: %v", err)
// 	}

// 	// Check number of files in the db
// 	rootCmd.Find([]string{"ls"})
// 	b1 = bytes.NewBufferString("")
// 	rootCmd.SetOut(b1)
// 	rootCmd.SetArgs([]string{"ls"})
// 	err = rootCmd.Execute()
// 	// if we encounter more than 1 newline, then we have more than 1 file (error)
// 	if err != nil || bytes.Count(b1.Bytes(), []byte("\n")) > 1 {
// 		t.Errorf("Error executing command: %v", err)
// 	}

// }

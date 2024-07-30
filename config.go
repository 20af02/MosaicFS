package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/20af02/MosaicFS/crypto"
	"github.com/20af02/MosaicFS/p2p"
	"github.com/joho/godotenv"
)

type NodeConfig struct {
	ListenAddr     string   `json:"listen_addr"`
	BootstrapNodes []string `json:"bootstrap_nodes"`
	ServerID       string   `json:"server_id"`
	EncKey         []byte
	DBFile         string
}

const envDir = "./.env" // Directory to store .env files

// loadConfig loads the node configuration from the .env file in the specified directory.
func loadConfig(envDir, listenAddr string) (*NodeConfig, error) {
	envFile := filepath.Join(envDir, fmt.Sprintf("server_%s.env", listenAddr[1:]))

	// Clear MOSAICFS_DB_FILE environment variable
	os.Unsetenv("MOSAICFS_DB_FILE")
	os.Unsetenv("MOSAICFS_ENC_KEY")
	os.Unsetenv("MOSAICFS_SERVER_ID")

	err := godotenv.Load(envFile)
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	serverID := os.Getenv("MOSAICFS_SERVER_ID")
	if serverID == "" {
		return nil, errors.New("MOSAICFS_SERVER_ID environment variable not set")
	}

	encKeyStr := os.Getenv("MOSAICFS_ENC_KEY")
	if encKeyStr == "" {
		return nil, errors.New("MOSAICFS_ENC_KEY environment variable not set")
	}

	encKey, err := hex.DecodeString(encKeyStr)
	if err != nil {
		return nil, fmt.Errorf("error decoding encryption key: %w", err)
	}

	dbFile := os.Getenv("MOSAICFS_DB_FILE")
	if dbFile == "" {
		return nil, errors.New("MOSAICFS_DB_FILE environment variable not set")
	}
	fmt.Printf("dbFile: %s\n", dbFile)

	return &NodeConfig{
		ListenAddr:     listenAddr,
		BootstrapNodes: []string{}, // Load this from the main config file
		ServerID:       serverID,
		EncKey:         encKey,
		DBFile:         dbFile,
	}, nil
}

// saveConfig saves the node configuration to a .env file in the specified directory.
func (c *NodeConfig) saveConfig(envDir string) error {
	envFile := filepath.Join(envDir, fmt.Sprintf("server_%s.env", c.ListenAddr[1:]))

	content := fmt.Sprintf("MOSAICFS_SERVER_ID=%s\nMOSAICFS_ENC_KEY=%s\nMOSAICFS_DB_FILE=%s", c.ServerID, hex.EncodeToString(c.EncKey), c.DBFile)
	return os.WriteFile(envFile, []byte(content), 0600)

}

// EnsureEnvDirExists creates the .env directory if it doesn't exist.
func EnsureEnvDirExists(envDir string) error {
	if _, err := os.Stat(envDir); os.IsNotExist(err) {
		return os.Mkdir(envDir, 0755) // Create directory with read/write/execute permissions
	}
	return nil
}

// loadConfig loads the node configurations from a JSON file.
func loadNodeConfig(filePath string) ([]NodeConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var nodeConfigs []NodeConfig
	if err := json.Unmarshal(data, &nodeConfigs); err != nil {
		return nil, fmt.Errorf("error parsing main config: %w", err)
	}
	// log.Printf("Loaded %d node configurations from JSON\n", len(nodeConfigs))
	return nodeConfigs, nil
}

// fileExists checks if a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	return !os.IsNotExist(err) && !info.IsDir()
}

// loadOrCreateConfig handles loading or initializing the NodeConfig
func loadOrCreateConfig(envDir string, baseConfig *NodeConfig) (*NodeConfig, error) {
	loadedConfig, err := loadConfig(envDir, baseConfig.ListenAddr)
	if err == nil {
		loadedConfig.BootstrapNodes = baseConfig.BootstrapNodes
		// Handle config creation (error or not found)
		if baseConfig.ServerID == "" {
			if len(loadedConfig.ListenAddr) == 0 {

				baseConfig.ServerID = crypto.GenerateID()
			} else {
				baseConfig.ServerID = loadedConfig.ServerID
			}
		}
		if len(baseConfig.EncKey) == 0 {
			if len(loadedConfig.EncKey) == 0 {
				baseConfig.EncKey = crypto.NewEncryptionKey()
			} else {
				baseConfig.EncKey = loadedConfig.EncKey
			}
		}
		if len(baseConfig.DBFile) == 0 || !fileExists(baseConfig.DBFile) {
			baseConfig.DBFile = filepath.Join(envDir, "db", fmt.Sprintf("server_%s.db", baseConfig.ListenAddr[1:]))
		}
		return loadedConfig, nil // Successfully loaded from file
	}

	// Handle config creation (error or not found)
	if baseConfig.ServerID == "" {

		if len(loadedConfig.ListenAddr) == 0 {
			baseConfig.ServerID = crypto.GenerateID()
		} else {
			baseConfig.ServerID = loadedConfig.ServerID
		}
	}
	if len(baseConfig.EncKey) == 0 {
		if len(loadedConfig.EncKey) == 0 {
			baseConfig.EncKey = crypto.NewEncryptionKey()
		} else {
			baseConfig.EncKey = loadedConfig.EncKey
		}
	}
	if len(baseConfig.DBFile) == 0 || !fileExists(baseConfig.DBFile) {
		baseConfig.DBFile = filepath.Join(envDir, "db", fmt.Sprintf("server_%s.db", baseConfig.ListenAddr[1:]))
	}

	if err := baseConfig.saveConfig(envDir); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}
	fmt.Printf("[%s] Config created: %s\n", baseConfig.ListenAddr, baseConfig.ServerID)
	fmt.Printf("BS: %v\n", baseConfig.BootstrapNodes)

	return baseConfig, nil
}

func makeServer(initialConfig *NodeConfig) *FileServer {
	if err := EnsureEnvDirExists(envDir); err != nil {
		log.Fatalf("Error creating .env directory: %s", err)
	}

	// Load or create node config

	nodeConfig, err := loadOrCreateConfig(envDir, initialConfig)
	if err != nil {
		return nil
	}
	log.Printf("[%s] Config loaded/created", nodeConfig.ListenAddr)

	// 3. Create TCP Transport
	tcpTransport := p2p.NewTCPTransport(p2p.TCPTransportOpts{
		ListenAddr:    nodeConfig.ListenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		//   OnPeer: 	OnPeer,
	})

	// 4. Create FileServer
	fileServer := NewFileServer(FileServerOpts{
		ID:                nodeConfig.ServerID,
		EncKey:            nodeConfig.EncKey,
		StorageRoot:       nodeConfig.ListenAddr[1:] + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootStrapNodes:    nodeConfig.BootstrapNodes,
		DBFile:            nodeConfig.DBFile,
	})

	tcpTransport.OnPeer = fileServer.OnPeer

	return fileServer
}

// createAndStartServers creates FileServer instances based on the configurations and starts them concurrently.
func createAndStartServers(configs []NodeConfig) ([]*FileServer, error) {
	var servers []*FileServer

	for _, config := range configs {
		servers = append(servers, makeServer(&config))
	}
	log.Printf("Created %d servers\n", len(servers))

	for _, server := range servers {
		go func(s *FileServer) {
			if err := s.Start(); err != nil {
				log.Fatalf("Error starting server: %s", err)
			}
		}(server)
		// time.Sleep(5 * time.Second)
	}

	return servers, nil
}

// New function to load and create servers from config file
func LoadAndCreateServers(filePath string, bootstrapNodes ...string) ([]*FileServer, error) {

	var nodeConfigs []NodeConfig

	nodeConfigs, err := loadNodeConfig(filePath)
	if err != nil || len(nodeConfigs) == 0 {
		node := bootstrapNodes[0]
		// trim whitespace
		node = strings.Trim(node, " ")
		// 0.0.0.0:3000 => :3000
		node = ":" + strings.Split(node, ":")[1]

		nodeConfig, err := loadConfig(envDir, node)
		if err != nil {
			return nil, err
		}

		for _, tnode := range bootstrapNodes[1:] {
			if tnode != node {

				nodeConfig.BootstrapNodes = append(nodeConfig.BootstrapNodes, strings.Trim(tnode, " "))
			}
		}
		fmt.Printf("Bootstrap nodes: %v\n", nodeConfig.BootstrapNodes)
		nodeConfigs = append(nodeConfigs, *nodeConfig)
	}

	return createAndStartServers(nodeConfigs)
}

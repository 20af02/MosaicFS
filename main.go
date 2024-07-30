package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	configFile := flag.String("config", "", "Path to the configuration file")
	bootstrapNodes := flag.String("nodes", ":3000, :4000, :5000", "Comma separated list of ports (e.g. :3000, :4000, :5000). The first is the master node, the rest are called to join the network.")
	flag.Parse()

	if err := initializeDirectories(bootstrapNodes); err != nil {
		log.Fatal(err)
	}

	nodeList := strings.Split(*bootstrapNodes, ",")

	servers, err := LoadAndCreateServers(*configFile, nodeList...)
	if err != nil {
		log.Fatal(err)
	}
	// app is the first server in the list
	app := servers[0]

	fsCli := NewFileServerCLI(app)
	Tui(fsCli)

}

func initializeDirectories(bootstrapNodes *string) error {
	ports := strings.Split(*bootstrapNodes, ",")
	for i, p := range ports {
		ports[i] = strings.TrimSpace(p) // Clean up whitespace
		if ports[i] == "" {
			log.Fatal("Invalid port format. Please use a comma-separated list (e.g., :3000,:4000,:5000)")
		}
	}

	if err := os.MkdirAll(".env", 0755); err != nil {
		return fmt.Errorf("error creating .env directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(".env", "db"), 0755); err != nil {
		return fmt.Errorf("error creating .env/db directory: %w", err)
	}

	for _, portStr := range ports {
		port := strings.TrimPrefix(portStr, ":")
		envFile := filepath.Join(".env", fmt.Sprintf("server_%s.env", port))
		dbFile := filepath.Join(".env", "db", fmt.Sprintf("server_%s.db", port))
		lockFile := filepath.Join(".env", "db", fmt.Sprintf("server_%s.db.lock", port))

		for _, file := range []string{envFile, dbFile, lockFile} {
			if _, err := os.Stat(file); os.IsNotExist(err) {
				if _, err := os.Create(file); err != nil {
					return fmt.Errorf("error creating file %s: %w", file, err)
				}
			}
		}
	}
	return nil
}

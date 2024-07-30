package main

import (
	"flag"
	"log"
	"strings"
)

func main() {
	configFile := flag.String("config", "", "Path to the configuration file")
	bootstrapNodes := flag.String("nodes", ":3000, :4000, :5000", "Comma separated list of ports (e.g. :3000, :4000, :5000). The first is the master node, the rest are called to join the network.")
	flag.Parse()

	// "node1:3000, node2:4000, node3:5000"
	// => ["node1:3000", "node2:4000", "node3:5000"]

	nodeList := strings.Split(*bootstrapNodes, ",")

	servers, err := LoadAndCreateServers(*configFile, nodeList...)
	if err != nil {
		log.Fatal(err)
	}
	// app is the first server in the list
	app := servers[0]

	fsCli := NewFileServerCLI(app)
	// go func() {
	Tui(fsCli)
	// }()

	// time.Sleep(15 * time.Second)

	// select {}
}

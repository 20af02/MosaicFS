# MosaicFS 

MosaicFS is a decentralized, content-addressable file system built in Go. It empowers you to securely store and share files across a peer-to-peer network without relying on central servers.

# 1. Intro
## Key Features
* **Decentralized Storage:** Your files are distributed across multiple nodes, ensuring no single point of failure.
* **Content Integrity:** Files are identified by their unique content (hash), guaranteeing they haven't been tampered with.
* **Efficient Large File Handling:** Splitting files into chunks enables fast, reliable transfers across the network.
* **Customizable P2P Network:** Tailor the communication layer to your specific needs.
* **Namespace Isolation:** Isolated data storage for enhanced privacy.

## Why Choose MosaicFS?

MosaicFS is perfect for:
* **Secure Data Sharing:** Collaborate on projects without worrying about data leaks.
* **Building dApps:** Create decentralized applications with reliable, censorship-resistant storage.
* **Archiving Content:** Preserve valuable information for the long term.

# 2. Getting Started
Prerequisites
- Go (version 1.22 or later)
- Docker (optional, for containerized deployment)


## Build
1. Clone the repository:
```bash
git clone https://github.com/20af02/MosaicFS.git
```
2. Build the project:
```bash
make build
```
3. (Optional) Build the Docker image:
```bash
make compose-build
```

# 3.  Usage
## 1. Creating multiple nodes

### Without Docker
-  Using the config file (eg. [config.json](https://github.com/20af02/MosaicFS/config.json))
```json
[
      {
            "listen_addr": ":3000",
            "bootstrap_nodes": []
      },
      {
            "listen_addr": ":4000",
            "bootstrap_nodes": ["localhost:3000"]
      },
      {
            "listen_addr": ":5000",
            "bootstrap_nodes": ["localhost:3000", "localhost:4000"]
      }
]
```
```bash
make run-multi
```
This creates three nodes on ports `3000`, `4000`, and `5000`. The bootstrap nodes specify the initial nodes to connect to.


- Passing the nodes as arguments
```bash
make build
./bin/fs -nodes :3000
./bin/fs -nodes :4000,:3000,
./bin/fs -nodes :5000,:3000,:4000
# ./bin/fs -nodes :<new_node_port>,:<listen_port1>,:<listen_port2>...
```
By default, make-run only creates one node on port `3000`.

### Docker Compose (Recommended)
Modify the [docker-compose.yml](https://github.com/20af02/MosaicFS/docker-compose.yml) file to specify the number of nodes and their configurations:

```yaml
services:
  node1:
    build: .
    container_name: mosaicfs-node1
    command: ["./bin/fs", "-nodes", "0.0.0.0:<node1_port>"]
    volumes:
      - ./<node1_port>_network:/app/<node1_port>_network
      - .env/server_<node1_port>.env:/app/.env/server_<node1_port>.env
      - .env/db/server_<node1_port>.db:/app/.env/db/server_<node1_port>.db
      - .env/db/server_<node1_port>.db.lock:/app/.env/db/server_<node1_port>.db.lock
      - .env/.db:/app/.env/.db
    networks:
      - mosaic-network 
    env_file:
      - .env/server_<node1_port>.env 
    tty: true
    stdin_open: true  

  node2:
    build: .
    container_name: mosaicfs-node2
    command: ["./bin/fs", "-nodes", "0.0.0.0:<node2_port>,node1:<node1_port>"]
    depends_on:
      - node1
    volumes:
      - ./<node2_port>_network:/app/<node2_port>_network
      - .env/server_<node2_port>.env:/app/.env/server_<node2_port>.env
      - .env/db/server_<node2_port>.db:/app/.env/db/server_<node2_port>.db
      - .env/db/server_<node2_port>.db.lock:/app/.env/db/server_<node2_port>.db.lock
      - .env/.db:/app/.env/.db
    networks:
      - mosaic-network
    env_file:
      - .env/server_<node2_port>.env 
    tty: true
    stdin_open: true  

networks:
  mosaic-network: 
    driver: bridge
```

Then start the containers:
```bash
make up
```

When finished, tear down the containers:

```bash
make down
```

## 2. Code Example: Storing a File
```go
package main

import (
	"context"
	"log"
  "github.com/20af02/MosaicFS"
  )

func main() {
	// 1. Create Server Instances
	server1 := makeServer(&NodeConfig{
		ListenAddr:     ":3000",
		BootstrapNodes: []string{},
	})
	server2 := makeServer(&NodeConfig{
		ListenAddr:     ":4000",
		BootstrapNodes: []string{"localhost:3000"},
	})

	// 2. Start Servers in Background
	go log.Fatal(server1.Start())
	go log.Fatal(server2.Start())
	defer server1.Stop()
	defer server2.Stop()

	time.Sleep(1 * time.Second)

	// 3. Store a file
	key := "my_important_file.txt"
	file, err := os.Open(key)
	if err != nil {
		log.Printf("Error opening file: %s", err)
		return
	}
	defer file.Close()

	if err := server1.Store(key, file); err != nil {
		log.Fatalf("Error storing file: %s", err)
	}

	log.Println("File stored successfully!")
}
```

# 4. TUI Interface
Interact with MosaicFS using the command-line interface:

```bash
# store a file (specific to the current node's namespace, contents are hashed and replicated across the network. Other nodes cannot directly access this file)
mosaicfs store <your_file>

# Get a file (locally or from the network specific to the current node's namespace)
mosaicfs get <file_name>

# Delete a file (locally or from the network specific to the current node's namespace)
mosaicfs delete --local <file_name>

# List stored files, and thier last known replica locations for the current node's namespace
mosaicfs ls 
```



For more commands and options, run ```help```.

## 1. Connecting to Containers 

```bash
docker attach <container_id> # (from docker ps. e.g., docker attach mosaicfs-node2)
```
```bash
make up-node # connects to mosaicfs-node2 by default
```


## 2. Example Usage
This example demonstrates how to use MosaicFS to store, retrieve, and delete the `README.md` file across multiple nodes in the network:

1. Start the MosaicFS containers:
```bash
make up
docker attach mosaicfs-node1
```
2. Inside the container:
```bash
store README.md 
# [:3000] received and written: (6866) bytes
ls
# File       Size (bytes)  Replicas  Locations
# README.md  6866          3         [:3000 172.19.0.3:45724 172.19.0.4:60648]
delete --local README.md 
# Local file [README.md] deleted successfully!
ls
# File       Size (bytes)  Replicas  Locations
# README.md  6866          2         [172.19.0.3:45724 172.19.0.4:60648]
get README.md 
# File [README.md] retrieved successfully!
ls
# File       Size (bytes)  Replicas  Locations
# README.md  6866          3         [:3000 172.19.0.3:45724 172.19.0.4:60648]
delete README.md 
# [README.md] deleted successfully!
ls
# File       Size (bytes)  Replicas  Locations
```


# 5. Contributing

Contributions are welcome! Whether you're interested in improving the network library, enhancing the CLI, or adding new features, your help is valued. Please follow these steps to contribute:

First fork the repository, then clone it to your local machine:
```bash
git clone https://github.com/<YOUR-USERNAME>/MosaicFS.git
cd MosaicFS
# Create a new branch
git checkout -b feature/YourFeature
# Commit your changes
git commit -am 'Add some feature'
# Push to the branch
git push origin feature/YourFeature
```
Open a pull request with a detailed description of your changes.


## Licensing
The source code in this repository and resulting container images are licensed under the [MIT License](https://opensource.org/license/mit). Feel free to use, modify, and distribute the project as you see fit.

## Disclaimer:
MosaicFS is still in active development and may contain bugs or limitations. Use it at your own risk. We are not responsible for any data loss or other issues that may arise.
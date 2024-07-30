# MosaicFS A Decentralized, Content-Addressable File System in Go

MosaicFS is a , open-source file storage system designed for the decentralized web. Built with Go, it prioritizes data integrity, scalability, and efficient handling of large files through content addressing and a custom peer-to-peer network.

# 1. Intro
## Overview
MosaicFS is a framework for building distributed storage solutions without the need for central servers. By combining content addressing with a custom-built peer-to-peer library, it offers a unique blend of reliability, efficiency, and customizability.

## Key Features

- __Content Addressing__: Files are uniquely identified by their content (hash), ensuring data integrity and deduplication.

- __Peer-to-Peer Architecture__: Files are distributed across a network of nodes, providing fault tolerance and scalability.

- __Large File Streaming__: Efficiently handles massive files by splitting them into chunks and streaming them across the network.

- __Custom P2P Network (TCP-based)__: A flexible and optimized peer-to-peer library designed for fast and reliable communication.

- __CLI Interface (Cobra)__: Intuitive command-line interface for managing files (`store`, `get`, `list`, etc.).
- __Dockerized Deployment__: Easily spin up nodes using Docker containers for quick setup and scalability.

## Why Choose MosaicFS?

MosaicFS is ideal for applications that require:

- __Decentralized Storage__: Distribute your data across multiple nodes without relying on a central server.
- __Namespace Isolation__: Store files in a specific namespace that is only accessible to your node, while still being replicated across the network.
- __Content Integrity__: Ensure your files remain unchanged and verifiable through content addressing.
- __Efficient Large File Handling__: Break down and distribute large files seamlessly across the network.
- __Customizability__: Leverage the flexible peer-to-peer library to tailor communication protocols for your specific needs.

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

3. Build the container image (Recommended):
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
This creates three nodes on ports `3000`, `4000`, and `5000`. The third node knows about the first two nodes, and the second node knows about the first node.


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
      - node2
    volumes:
      - ./<node2_port>_network:/app/<node2_port>_network
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

## Connecting to Containers 

```bash
docker attach <container_id> # (from docker ps. e.g., docker attach mosaicfs-node2)
```
```bash
make up-node # connects to mosaicfs-node2 by default
```


## 2. Example Usage
This basic example stores this `README.md` file in the network, deletes it locally, then fetches it, and finally deletes it from the network.

```bash
make up
docker attach mosaicfs-node1
```
Inside the container:
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
MosaicFS is under active development and may have limitations or bugs. Use at your own risk.

# 6. Release Notes

## v1.0.0 (2024-07-29)
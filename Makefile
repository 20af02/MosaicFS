build:
	@go build -o bin/fs

run: build
	@./bin/fs 

run-multi:
	@./bin/fs -config ./config.json

test:
	@go test -v ./...


# Docker Compose targets
COMPOSE=docker-compose
SERVICE=mosaicfs

up:
	$(COMPOSE) up --build -d

up-node: up

	docker attach $(SERVICE)-node2

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

# New Exec function 
exec-node:
	$(COMPOSE) exec $(NODE) sh

# Build from compose file
compose-build:
	$(COMPOSE) build


# Clean targets
clean: clean-docker clean-network

clean-docker:
	$(COMPOSE) down --rmi all --volumes --remove-orphans
	docker network prune -f    
	docker volume prune -f  
           
clean-network: 
	ifeq ($(OS),Windows_NT)
		del /s /q *_network
	else
		rm -rf *_network
	endif
build:
	@go build -o bin/fs

run: build
	@./bin/fs/MosaicFS.exe
	# @./bin/fs

test:
	@go test -v ./...
# Use the official Go base image
FROM golang:1.22.5

# Set the working directory within the container
WORKDIR /app

# Copy go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download project dependencies
RUN go mod download

# Copy the entire project source code
COPY . .

# Use the Makefile to build the Go binary 
RUN make build

# Expose the ports the application listens on 
# EXPOSE 3000 4000 5000

# Command to run the application when the container starts
# CMD ["./bin/fs"] 

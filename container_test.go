package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestDockerComposeUpAttachAndBuild(t *testing.T) {
	// 1. Build the Docker image (if not already built)
	cmd := exec.Command("make", "compose-build")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build Docker image: %s\n%s", err, output)
	}

	// 2. Bring up Docker Compose environment
	cmd = exec.Command("make", "up")
	err = cmd.Start() // Start in the background
	if err != nil {
		t.Fatalf("Failed to start Docker Compose: %s", err)
	}

	// 3. Wait for containers to stabilize
	time.Sleep(5 * time.Second)

	// 4. Read logs from a container
	cmd = exec.Command("docker", "logs", "mosaicfs-node2")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to attach to container: %s\n%s", err, output)
	}

	// 5. Verification steps:
	nodes := []string{"node2", "node3", "node4"}
	expectedLog := "starting server..."       // Or another expected log message
	connectedLog := "Connected with remote: " // Or another expected log message for connection verification
	for _, node := range nodes {
		cmd := exec.Command("docker", "logs", fmt.Sprintf("mosaicfs-%s", node))
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get logs for container mosaicfs-%s: %s\n%s", node, err, output)
		}

		// Check for expected log messages
		if !strings.Contains(string(output), expectedLog) {
			t.Errorf("Expected log message '%s' not found in mosaicfs-%s logs:\n%s", expectedLog, node, output)
		}
		if !strings.Contains(string(output), connectedLog) {
			t.Errorf("Expected log message '%s' not found in mosaicfs-%s logs:\n%s", connectedLog, node, output)
		}
	}

	// 7. Stop and remove the containers
	cmd = exec.Command("make", "down")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to tear down Docker Compose: %s\n%s", err, output)
	}
}

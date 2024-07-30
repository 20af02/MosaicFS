package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NewFileServerCLI creates a new Cobra command for interacting with the FileServer.
func NewFileServerCLI(fs *FileServer) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "mosaicfs",
		Short: "MosaicFS CLI",
	}

	// get Command
	var getNode string
	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get a file from the network",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]

			_, err := fs.Get(key)
			if err != nil {
				log.Fatalf("Error getting file: %v", err)
			}
			fmt.Printf("File [%s] retrieved successfully!\n", key)
			// fmt.Printf("File contents:\n %s\n", file)

		},
	}
	getCmd.Flags().StringVarP(&getNode, "node", "n", fs.Transport.Addr(), "Node address to fetch from")

	// store Command
	storeCmd := &cobra.Command{
		Use:   "store [filepath]",
		Short: "Store a file on the network",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			file, err := os.Open(filePath)
			if err != nil {
				log.Printf("Error opening file: %s", err)
				return
			}
			defer file.Close()

			key := filePath // Or use a different key generation strategy
			if err := fs.Store(key, file); err != nil {
				log.Fatalf("Error storing file: %s", err)
			}
		},
	}

	// delete Command
	deleteCmd := &cobra.Command{
		Use:   "delete [key]",
		Short: "Delete a file from the network",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]

			// 1. Get flag value
			deleteLocal, err := cmd.Flags().GetBool("local") // Add --local flag
			if err != nil {
				fmt.Printf("Error getting --local flag: %s\n", err)
				return
			}

			// 2. Check if the flag is set
			if deleteLocal {
				if err := fs.store.Delete(fs.ID, key); err != nil {
					fmt.Printf("Error deleting local file [%s]: %s\n", key, err)
					return
				}
				// Remove the local file from db
				fmt.Printf("Local file [%s] deleted successfully!\n", key)
				return
			}
			if err := fs.Delete(key); err != nil {
				fmt.Printf("Error deleting file [%s]: %s", key, err)
				return
			}
			fmt.Printf("[%s] deleted successfully!\n", key)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Reset flag to its default value before each run
			err := cmd.Flags().Set("local", "false")
			return err // Propagate any errors from Set
		},
	}
	deleteCmd.Flags().BoolP("local", "l", false, "Delete the file locally")

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List all files on the network",
		Run: func(cmd *cobra.Command, args []string) {
			files, err := fs.ListFiles()
			if err != nil {
				log.Fatalf("Error listing files: %v", err)
			}
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			// Add rows to the tabwriter
			fmt.Fprintln(w, "File\tSize (bytes)\tReplicas\tLocations") // Header
			for _, file := range files {
				fmt.Fprintf(w, "%s\t%d\t%d\t%v\n", file.Key, file.Size, file.Replicas, file.ReplicaLocations)
			}

			// Flush the tabwriter's buffer to output
			w.Flush()
		},
	}

	rootCmd.AddCommand(getCmd, storeCmd, deleteCmd, lsCmd)

	return rootCmd
}

func Tui(rootCmd *cobra.Command) {
	// Create a Promptui prompt
	prompt := promptui.Prompt{
		Label: "mosaicfs > ",
		Validate: func(input string) error {
			if len(input) == 0 {
				return errors.New("please enter a command")
			}
			return nil
		},
		Stdin: os.Stdin,
	}
	// log.Println("Exiting MosaicFS CLI...")
	defer log.Println("Exited MosaicFS CLI...")

	// Interactive CLI Loop with Promptui
	for {
		input, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				break // Exit the loop on Ctrl+C
			}

			fmt.Fprintln(os.Stderr, err)
			continue
		}

		// Parse and execute the command
		cmdArgs := strings.Fields(input)

		if len(cmdArgs) > 0 {
			_, _, err := rootCmd.Find(cmdArgs) // Use the cli command
			if err != nil {
				fmt.Fprintln(os.Stderr, "Invalid command:", err)
				continue
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			rootCmd.SetArgs(cmdArgs)
			if err := rootCmd.ExecuteContext(ctx); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}

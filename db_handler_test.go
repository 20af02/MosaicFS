package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDBHandler(t *testing.T) {
	dbFile := createTempDBFile(t)
	defer os.Remove(dbFile)

	dh, err := NewDBHandler("server1", dbFile)
	if err != nil {
		t.Fatal(err)
	}
	defer dh.Close()

	// 1. Test UpdateFile (and initial GetFileMetadata)
	fmd := FileMetadata{Key: "test.txt", Size: 1024, Replicas: 1, ReplicaLocations: []string{"127.0.0.1"}}
	added, err := dh.UpdateFile(fmd)
	require.NoError(t, err, "UpdateFile should succeed")
	require.True(t, added, "File should be newly added")

	// 2. Test GetFileMetadata (after update)
	storedFmd, err := dh.GetFileMetadata("test.txt")
	require.NoError(t, err, "GetFileMetadata should succeed")
	require.True(t, compareFileMetadata(storedFmd, &fmd),
		"Stored metadata should match the original")

	// 3. Test ListFiles (should have one entry now)
	files, err := dh.ListFiles()
	require.NoError(t, err, "ListFiles should succeed")
	require.Len(t, files, 1, "Should have one file listed")
	require.True(t, compareFileMetadata(&files[0], &fmd),
		"Listed metadata should match")

	// 4. Test DeleteFileMetadata
	err = dh.DeleteFileMetadata("test.txt")
	require.NoError(t, err, "DeleteFileMetadata should succeed")

	// 5. Test GetFileMetadata (after delete, should fail)
	_, err = dh.GetFileMetadata("test.txt")
	require.Error(t, err, "GetFileMetadata should fail after deletion")

	// 6. Test ListFiles (should be empty again)
	files, err = dh.ListFiles()
	require.NoError(t, err, "ListFiles should succeed")
	require.Empty(t, files, "List should be empty after deletion")
}

// Helper function to create a temporary database file for testing
func createTempDBFile(t *testing.T) string {
	f, err := os.CreateTemp("", "test_db_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	return f.Name()
}

// Helper function to compare two FileMetadata structs
func compareFileMetadata(fmd1, fmd2 *FileMetadata) bool {
	return fmd1.Key == fmd2.Key && fmd1.Size == fmd2.Size && fmd1.Replicas == fmd2.Replicas &&
		equalStringSlices(fmd1.ReplicaLocations, fmd2.ReplicaLocations)
}

// Helper function to compare string slices
func equalStringSlices(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

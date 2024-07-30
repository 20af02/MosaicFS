package main

import (
	// For temporary file creation
	"os"
	"testing"
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
	if err != nil {
		t.Errorf("UpdateFile failed: %v", err)
	}
	if !added {
		t.Error("Expected file to be added")
	}

	// Since the DB was empty, this should be the first entry.
	storedFmd, err := dh.GetFileMetadata("test.txt")
	if err != nil {
		t.Errorf("GetFileMetadata (after update) failed: %v", err)
	}
	if !compareFileMetadata(storedFmd, &fmd) {
		t.Error("Stored metadata does not match the original")
	}

	// 2. Test GetFileMetadata (after update)
	storedFmd, err = dh.GetFileMetadata("test.txt")
	if err != nil {
		t.Errorf("GetFileMetadata failed: %v", err)
	}
	if !compareFileMetadata(storedFmd, &fmd) {
		t.Error("Stored metadata does not match the expected value")
	}

	// 3. Test ListFiles (should have one entry now)
	files, err := dh.ListFiles()
	if err != nil {
		t.Errorf("ListFiles failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
	if !compareFileMetadata(&files[0], &fmd) {
		t.Error("Listed metadata does not match the expected value")
	}

	// 4. Test DeleteFileMetadata
	err = dh.DeleteFileMetadata("test.txt")
	if err != nil {
		t.Errorf("DeleteFileMetadata failed: %v", err)
	}

	// 5. Test GetFileMetadata (after delete, should fail)
	_, err = dh.GetFileMetadata("test.txt")
	if err == nil {
		t.Error("GetFileMetadata should have returned an error after deletion")
	}

	// 6. Test ListFiles (should be empty again)
	files, err = dh.ListFiles()
	if err != nil {
		t.Errorf("ListFiles failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
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

package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestPathTransformFunc(t *testing.T) {
	key := "awesomePicture"
	pathKey := CASPathTransformFunc(key)
	expectedOriginalKey := "34ae125e927e54823ca1d7c8726e3a8c273de692"
	expectedPathName := "34ae1/25e92/7e548/23ca1/d7c87/26e3a/8c273/de692"

	if pathKey.PathName != expectedPathName {
		t.Errorf("Expected: %s Actual: %s", expectedPathName, pathKey.PathName)
	}
	if pathKey.Filename != expectedOriginalKey {
		t.Errorf("Expected: %s Actual: %s", expectedOriginalKey, pathKey.Filename)
	}
}

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	store := NewStore(opts)
	key := "awesomePicture"
	data := []byte("some bytes data")
	// bytes.NewReader([]byte("some bytes data"))

	if err := store.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Errorf("Failed to writeStream: %v", err)
	}
	// wait

	// time.Sleep(5 * time.Second)

	if err := store.Delete(key); err != nil {
		t.Errorf("Failed to delete key: %v", err)
	}

}

func TestStore(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	store := NewStore(opts)
	key := "awesomePicture"
	data := []byte("some bytes data")
	// bytes.NewReader([]byte("some bytes data"))

	if err := store.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Errorf("Failed to writeStream: %v", err)
	}

	if ok := store.Has(key); !ok {
		t.Errorf("Expected to have key: %s", key)
	}

	r, err := store.Read(key)
	if err != nil {
		t.Errorf("Failed to readStream: %v", err)
	}

	b, _ := io.ReadAll(r)
	if !bytes.Equal(b, data) {
		t.Errorf("Expected: %s Actual: %s", string(data), string(b))
	}

	fmt.Printf("Data: %s\n", string(b))

}

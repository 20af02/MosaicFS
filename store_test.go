package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/20af02/MosaicFS/crypto"
)

func TestPathTransformFunc(t *testing.T) {
	key := "awesomePicture"
	pathKey := CASPathTransformFunc(key)

	expectedFileName := "34ae125e927e54823ca1d7c8726e3a8c273de692"
	expectedPathName := "34ae1\\25e92\\7e548\\23ca1\\d7c87\\26e3a\\8c273\\de692"

	if pathKey.PathName != expectedPathName {
		t.Errorf("Expected: %s Actual: %s", expectedPathName, pathKey.PathName)
	}
	if pathKey.Filename != expectedFileName {
		t.Errorf("Expected: %s Actual: %s", expectedFileName, pathKey.Filename)
	}
}

func TestStore(t *testing.T) {
	store := newStore()
	id := crypto.GenerateID()
	defer teardown(t, store)
	defer func() {
		os.Remove("./.env/.db/test.db")
	}()
	defer store.dbHandler.Close()

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("foo_%d", i)
		// "awesomePicture"
		data := []byte("some bytes data")
		// bytes.NewReader([]byte("some bytes data"))

		if _, err := store.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Errorf("Failed to writeStream: %v", err)
		}

		if ok := store.Has(id, key); !ok {
			t.Errorf("Expected to have key: %s", key)
		}

		_, r, err := store.Read(id, key)
		if err != nil {
			t.Errorf("Failed to readStream: %v", err)
		}

		b, _ := io.ReadAll(r)
		if !bytes.Equal(b, data) {
			t.Errorf("Expected: [%s] Actual: [%s]", string(data), string(b))
		}

		if err := store.Delete(id, key); err != nil {
			t.Error(err)
		}

		if ok := store.Has(id, key); ok {
			t.Errorf("Expected to NOT have key: %s", key)
		}

	}

}

func TestStoreFT(t *testing.T) {
	s1 := MakeTestServer(":3000", []string{})
	s2 := MakeTestServer(":4000", []string{":3000"})
	s3 := MakeTestServer(":5000", []string{":3000", ":4000"})
	defer teardown(t, s1.store)
	defer teardown(t, s2.store)
	defer teardown(t, s3.store)

	go func() { s1.Start() }()
	time.Sleep(2 * time.Second)
	go func() { s2.Start() }()
	time.Sleep(2 * time.Second)
	go func() { s3.Start() }()
	time.Sleep(2 * time.Second)

	for i := 0; i < 20; i++ {
		key := fmt.Sprintf("picture_%d.png", i)
		data := bytes.NewReader([]byte("my big data file here!"))
		s3.Store(key, data)

		if err := s3.store.Delete(s3.ID, key); err != nil {
			t.Errorf("Failed to delete: %v", err)
		}
		// time.Sleep(5 * time.Millisecond)
		// for i := 0; i < 10; i++ {

		// 	data := bytes.NewReader([]byte("my big data file here!"))
		// 	s2.Store(fmt.Sprintf("myprivatedata_%d", i), data)
		// 	time.Sleep(5 * time.Millisecond)
		// }

		r, err := s3.Get(key)
		if err != nil {
			t.Errorf("Failed to get: %v", err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			t.Errorf("Failed to readAll: %v", err)
		}

		fmt.Println(string(b))
	}
}

func TestNetDelete(t *testing.T) {
	s1 := MakeTestServer(":3000", []string{})
	s2 := MakeTestServer(":4000", []string{":3000"})
	s3 := MakeTestServer(":5000", []string{":3000", ":4000"})
	defer teardown(t, s1.store)
	defer teardown(t, s2.store)
	defer teardown(t, s3.store)

	go func() { s1.Start() }()
	go func() { s2.Start() }()
	time.Sleep(2 * time.Second)

	go func() { s3.Start() }()
	time.Sleep(2 * time.Second)

	key := "picture_1.png"
	data := bytes.NewReader([]byte("my big data file here!"))
	s3.Store(key, data)

	if err := s3.Delete(key); err != nil {
		t.Errorf("Failed to delete: %v", err)
	}

}

func newStore() *Store {
	db, _ := NewDBHandler("test", "./.env/.db/test.db")
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
		dbHandler:         db,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Errorf("Failed to clear store: %v", err)
	}
}

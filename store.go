package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/20af02/MosaicFS/crypto"
)

// filename => awesomePicture.png
// path => transformFunc(filename) => ROOT/pubkey/path
const defaultRootFolderName = "ggnetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize

	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i+1)*blockSize
		paths[i] = hashStr[from:to]
	}
	return PathKey{
		PathName: strings.Join(paths, string(filepath.Separator)),
		Filename: hashStr,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	Filename string
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, string(filepath.Separator))
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) FullPath() string {
	return filepath.Join(p.PathName, p.Filename)
	// return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

type StoreOpts struct {
	// Root is the folder name of the root directory containing all the files/folders of the store.
	Root string

	PathTransformFunc PathTransformFunc
	dbHandler         *DBHandler
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		Filename: key,
	}
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := filepath.Join(s.Root, id, pathKey.FullPath())
	_, err := os.Stat(fullPathWithRoot)

	// log.Printf("Error: %v", err)
	return !errors.Is(err, os.ErrNotExist)
	// return err != fs.ErrNotExist
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)
	defer func() {
		log.Printf("Deleted [%s] from disk", pathKey.Filename)
	}()
	firstPathNameWithNS := filepath.Join(id, pathKey.FirstPathName())
	firstPathNameWithRoot := filepath.Join(s.Root, firstPathNameWithNS)
	log.Printf("Deleting [%s]", firstPathNameWithRoot)

	// log.Printf("Deleting metadata for [%s]", pathKey.Filename)

	if err := s.dbHandler.RemoveLocalMetadata(key); err != nil {
		log.Printf("Error deleting metadata: %v", err)
	}
	log.Printf("[%s] deleting [%s]", id, firstPathNameWithRoot)

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	// if s.Has(key) {
	// 	return nil
	// }
	return s.writeStream(id, key, r)
}

func (s *Store) WriteDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {

	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := crypto.CopyDecrypt(encKey, r, f)
	return int64(n), err
}

func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := filepath.Join(s.Root, id, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPath := pathKey.FullPath()
	fullPathWithRoot := filepath.Join(s.Root, id, fullPath)
	// log.Printf("Writing to_: %s", fullPathWithRoot)
	return os.Create(fullPathWithRoot)
}

func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)

}

// @FIXME: Instead of copying directly to a reader, we first copy this into a buffer. Maybe just return the File from the readStream?
func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	// return s.readStream(key)
	n, f, err := s.readStream(id, key)
	if err != nil {
		return 0, nil, err
	}

	// Copy the file with a reader if on windows

	if runtime.GOOS == "windows" {
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, f)
		if err != nil {
			return 0, nil, err
		}
		defer f.Close()
		return n, buf, nil
	}

	// avoid keeping x-lock on the file
	pr, pw := io.Pipe()

	// copy data and close the pipe when done
	go func() {
		// defer wg.Done()
		defer pw.Close()
		defer f.Close()

		_, err := io.Copy(pw, f)
		if err != nil {
			log.Printf("Error: %v", err)
		}
	}()
	return n, pr, nil

}

func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := filepath.Join(s.Root, id, pathKey.FullPath())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil

}

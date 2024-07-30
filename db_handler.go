package main

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/20af02/MosaicFS/crypto"
	"github.com/boltdb/bolt"
)

// DBHandler handles interactions with the BoltDB database.
type DBHandler struct {
	db       *bolt.DB
	serverID string
	envDir   string // For storing the .env file
}

type FileMetadata struct {
	Key              string
	Size             int64
	Replicas         int
	ReplicaLocations []string
}

// NewDBHandler creates a new DBHandler instance.
func NewDBHandler(serverID, dbFile string) (*DBHandler, error) {
	// Open the database file or create it if it doesn't exist
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	return &DBHandler{
		db:       db,
		serverID: serverID,
		envDir:   envDir}, nil
}

// Close closes the database connection.
func (dh *DBHandler) Close() error {
	return dh.db.Close()
}

// UpdateFile updates a file's information to the db based on its hash.
// This includes information like the file's hash, size, number of replicas, and the replicas' locations.
func (dh *DBHandler) UpdateFile(fmd FileMetadata) (bool, error) {
	hashedKey := crypto.HashKey(fmd.Key)

	var added bool
	err := dh.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(dh.serverID))
		if err != nil {
			return err
		}

		// Store the file metadata
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(fmd); err != nil {
			return err
		}
		err = bucket.Put([]byte(hashedKey), buf.Bytes())
		if err != nil {
			return err
		}
		added = true

		return nil
	})

	return added, err

}

func (dh *DBHandler) RemoveLocalMetadata(key string) error {
	var fmd *FileMetadata
	fmd, err := dh.GetFileMetadata(key)
	if err != nil && (err.Error() == "file not found" || err.Error() == "bucket not found") {
		return nil
	} else if err != nil {
		return err
	}

	fmd.Replicas--
	if fmd.Replicas < 0 {
		return fmt.Errorf("number of replicas is less than 0")
	} else if fmd.Replicas == 0 {
		return dh.DeleteFileMetadata(key)
	}
	fmd.ReplicaLocations = fmd.ReplicaLocations[1:]

	// Write
	added, err := dh.UpdateFile(*fmd)
	if !added {
		return fmt.Errorf("error updating file metadata")
	}
	return err
}

func (dh *DBHandler) AddLocalMetaDataToExistingKey(key string, addr string) error {
	var fmd *FileMetadata
	fmd, err := dh.GetFileMetadata(key)
	if err != nil && (err.Error() == "file not found" || err.Error() == "bucket not found") {
		return nil
	} else if err != nil {
		return err
	}

	fmd.Replicas++
	fmd.ReplicaLocations = append([]string{addr}, fmd.ReplicaLocations...)
	added, err := dh.UpdateFile(*fmd)
	if !added {
		return fmt.Errorf("error updating file metadata")
	}
	return err
}

// GetFileMetadata retrieves the metadata of a file from the database.
func (dh *DBHandler) GetFileMetadata(key string) (*FileMetadata, error) {
	hashedKey := crypto.HashKey(key)

	var fmd *FileMetadata
	err := dh.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(dh.serverID))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		data := bucket.Get([]byte(hashedKey))
		if data == nil {
			return fmt.Errorf("file not found")
		}

		buf := bytes.NewBuffer(data)
		if err := gob.NewDecoder(buf).Decode(&fmd); err != nil {
			return err
		}

		return nil
	})

	return fmd, err
}

// DeleteFileMetadata deletes the metadata of a file from the database.
func (dh *DBHandler) DeleteFileMetadata(key string) error {
	hashedKey := crypto.HashKey(key)

	fmt.Printf("Deleting file metadata for key: %s\n", key)

	err := dh.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(dh.serverID))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		return bucket.Delete([]byte(hashedKey))
	})

	return err
}

// ListFiles returns a list of all files stored in the database under the server's ID.
func (dh *DBHandler) ListFiles() ([]FileMetadata, error) {
	var files []FileMetadata

	err := dh.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(dh.serverID))
		if bucket == nil {
			// return fmt.Errorf("bucket not found")
			return nil
		}

		return bucket.ForEach(func(k, v []byte) error {
			buf := bytes.NewBuffer(v)
			var fmd FileMetadata
			if err := gob.NewDecoder(buf).Decode(&fmd); err != nil {
				return err
			}

			files = append(files, fmd)
			return nil
		})
	})

	return files, err
}

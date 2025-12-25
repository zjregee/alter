package storage

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	defaultDir      = ".alter"
	defaultBucket   = "alter"
	defaultFileName = "alter.db"
)

type database struct {
	db        *bolt.DB
	closeOnce sync.Once
}

var (
	instance *database
	initOnce sync.Once
)

func init() {
	initOnce.Do(func() {
		var isFirstTime bool
		var err error
		instance, isFirstTime, err = newDatabase()
		if err != nil {
			panic(err)
		}
		if isFirstTime {
			initStorage()
		}
	})
}

func Close() error {
	if instance == nil {
		return nil
	}

	var err error
	instance.closeOnce.Do(func() {
		if instance.db != nil {
			err = instance.db.Close()
		}
	})
	return err
}

func Get(key []byte) ([]byte, error) {
	if instance == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return instance.get(key)
}

func Put(key, value []byte) error {
	if instance == nil {
		return fmt.Errorf("database not initialized")
	}
	return instance.put(key, value)
}

func Delete(key []byte) error {
	if instance == nil {
		return fmt.Errorf("database not initialized")
	}
	return instance.delete(key)
}

func List(prefix []byte) (map[string][]byte, error) {
	if instance == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return instance.list(prefix)
}

func newDatabase() (*database, bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get home directory: %w", err)
	}

	alterDir := filepath.Join(homeDir, defaultDir)
	if err := os.MkdirAll(alterDir, 0755); err != nil {
		return nil, false, fmt.Errorf("failed to create .alter directory: %w", err)
	}

	dbPath := filepath.Join(alterDir, defaultFileName)

	_, err = os.Stat(dbPath)
	isFirstTime := os.IsNotExist(err)

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(defaultBucket))
		return err
	})
	if err != nil {
		_ = db.Close()
		return nil, false, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &database{
		db: db,
	}, isFirstTime, nil
}

func (d *database) get(key []byte) ([]byte, error) {
	var value []byte
	err := d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(defaultBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", defaultBucket)
		}
		v := bucket.Get(key)
		if v != nil {
			value = make([]byte, len(v))
			copy(value, v)
		}
		return nil
	})
	return value, err
}

func (d *database) put(key, value []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(defaultBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", defaultBucket)
		}
		return bucket.Put(key, value)
	})
}

func (d *database) delete(key []byte) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(defaultBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", defaultBucket)
		}
		return bucket.Delete(key)
	})
}

func (d *database) list(prefix []byte) (map[string][]byte, error) {
	result := make(map[string][]byte)
	err := d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(defaultBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", defaultBucket)
		}
		cursor := bucket.Cursor()
		if len(prefix) == 0 {
			for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
				key := make([]byte, len(k))
				value := make([]byte, len(v))
				copy(key, k)
				copy(value, v)
				result[string(key)] = value
			}
		} else {
			for k, v := cursor.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cursor.Next() {
				key := make([]byte, len(k))
				value := make([]byte, len(v))
				copy(key, k)
				copy(value, v)
				result[string(key)] = value
			}
		}
		return nil
	})
	return result, err
}

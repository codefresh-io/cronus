package backend

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/codefresh-io/cronus/pkg/types"
	log "github.com/sirupsen/logrus"
)

type (
	// BoltEventStore BoltDB store
	BoltEventStore struct {
		db *bolt.DB
	}
)

var events = []byte("events")

// NewBoltEventStore new BoldDB store
func NewBoltEventStore(file string) (types.EventStore, error) {
	db, err := setupDB(file)
	return &BoltEventStore{db}, err
}

func setupDB(file string) (*bolt.DB, error) {
	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("could not open db, %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(events)
		if err != nil {
			return fmt.Errorf("could not create events bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not set up buckets, %v", err)
	}

	log.Debug("setup db done")
	return db, nil
}

// StoreEvent store event record into BoltDB
func (b *BoltEventStore) StoreEvent(event types.Event) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		v, _ := json.Marshal(event)
		return bucket.Put([]byte(types.GetURI(event)), v)
	})
}

// DeleteEvent delete event record from BoltDB
func (b *BoltEventStore) DeleteEvent(uri string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		return bucket.Delete([]byte(uri))
	})
}

// GetEvent get event record
func (b *BoltEventStore) GetEvent(uri string) (*types.Event, error) {
	var event types.Event
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		v := bucket.Get([]byte(uri))
		if v == nil {
			return errors.New("cron event not found")
		}
		return json.Unmarshal(v, &event)
	})
	return &event, err
}

// GetAllEvents get all stored events
func (b *BoltEventStore) GetAllEvents() ([]types.Event, error) {
	all := make([]types.Event, 0)
	b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		bucket.ForEach(func(k, v []byte) error {
			var event types.Event
			err := json.Unmarshal(v, &event)
			if err != nil {
				return err
			}
			all = append(all, event)
			return nil
		})
		return nil
	})
	return all, nil
}

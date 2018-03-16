package backend

import (
	"encoding/json"
	"fmt"
	"io"

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
	log.WithField("store", file).Debug("starting BoltDB")
	db, err := setupDB(file)
	return &BoltEventStore{db}, err
}

func setupDB(file string) (*bolt.DB, error) {
	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		log.WithError(err).Error("failed to open file")
		return nil, fmt.Errorf("failed to open db, %v", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(events)
		if err != nil {
			log.WithError(err).Error("failed to create events bucket")
			return fmt.Errorf("failed to create events bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		log.WithError(err).Error("failed to setup db")
		return nil, fmt.Errorf("failed to set up db, %v", err)
	}

	log.Debug("setup db done")
	return db, nil
}

// BackupDB backup BoltDB database
func (b *BoltEventStore) BackupDB(w io.Writer) (int, error) {
	log.Debug("database backup")
	var size int
	err := b.db.View(func(tx *bolt.Tx) error {
		size = int(tx.Size())
		_, err := tx.WriteTo(w)
		return err
	})
	if err != nil {
		log.WithError(err).Error("failed to backup db")
	}
	return size, err
}

// StoreEvent store event record into BoltDB
func (b *BoltEventStore) StoreEvent(event types.Event) error {
	log.WithFields(log.Fields{
		"expression": event.Expression,
		"message":    event.Message,
	}).Debug("storing new event")
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		v, _ := json.Marshal(event)
		return bucket.Put([]byte(types.GetURI(event)), v)
	})
}

// DeleteEvent delete event record from BoltDB
func (b *BoltEventStore) DeleteEvent(uri string) error {
	log.WithField("uri", uri).Debug("deleting event from store")
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		v := bucket.Get([]byte(uri))
		if v == nil {
			log.WithField("uri", uri).Error("event not found")
			return types.ErrEventNotFound
		}
		return bucket.Delete([]byte(uri))
	})
}

// GetEvent get event record
func (b *BoltEventStore) GetEvent(uri string) (*types.Event, error) {
	var event types.Event
	log.WithField("uri", uri).Debug("getting event from store")
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		v := bucket.Get([]byte(uri))
		if v == nil {
			log.WithField("uri", uri).Error("event not found")
			return types.ErrEventNotFound
		}
		return json.Unmarshal(v, &event)
	})
	if err != nil {
		log.WithError(err).Error("failed to get event")
		return nil, err
	}
	return &event, nil
}

// GetAllEvents get all stored events
func (b *BoltEventStore) GetAllEvents() ([]types.Event, error) {
	log.Debug("getting all events from store")
	all := make([]types.Event, 0)
	b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		bucket.ForEach(func(k, v []byte) error {
			var event types.Event
			err := json.Unmarshal(v, &event)
			if err != nil {
				log.WithError(err).Error("failed to parse JSON")
				return err
			}
			all = append(all, event)
			return nil
		})
		return nil
	})
	return all, nil
}

// GetDBStats get number of records
func (b *BoltEventStore) GetDBStats() (int, error) {
	var records int
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(events)
		stats := bucket.Stats()
		records = stats.KeyN
		return nil
	})
	if err != nil {
		log.WithError(err).Error("failed to get db stats")
	}
	return records, err
}

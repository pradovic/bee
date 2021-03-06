// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package shed provides a simple abstraction components to compose
// more complex operations on storage data organized in fields and indexes.
//
// Only type which holds logical information about swarm storage chunks data
// and metadata is Item. This part is not generalized mostly for
// performance reasons.
package shed

import (
	"github.com/ethersphere/bee/pkg/logging"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	openFileLimit = 128 // The limit for LevelDB OpenFilesCacheCapacity.
)

// DB provides abstractions over LevelDB in order to
// implement complex structures using fields and ordered indexes.
// It provides a schema functionality to store fields and indexes
// information about naming and types.
type DB struct {
	ldb     *leveldb.DB
	metrics metrics
	logger  logging.Logger
	quit    chan struct{} // Quit channel to stop the metrics collection before closing the database
}

// NewDB constructs a new DB and validates the schema
// if it exists in database on the given path.
// metricsPrefix is used for metrics collection for the given DB.
func NewDB(path string, logger logging.Logger) (db *DB, err error) {
	ldb, err := leveldb.OpenFile(path, &opt.Options{
		OpenFilesCacheCapacity: openFileLimit,
	})
	if err != nil {
		return nil, err
	}
	db = &DB{
		ldb:     ldb,
		metrics: newMetrics(),
		logger:  logger,
	}

	if _, err = db.getSchema(); err != nil {
		if err == leveldb.ErrNotFound {
			// save schema with initialized default fields
			if err = db.putSchema(schema{
				Fields:  make(map[string]fieldSpec),
				Indexes: make(map[byte]indexSpec),
			}); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Create a quit channel for the periodic metrics collector and run it
	db.quit = make(chan struct{})

	return db, nil
}

// Put wraps LevelDB Put method to increment metrics counter.
func (db *DB) Put(key []byte, value []byte) (err error) {
	err = db.ldb.Put(key, value, nil)
	if err != nil {
		db.logger.Debugf("failed to insert in to DB. Error : %s",  err.Error())
		db.metrics.PutFailCounter.Inc()
		return err
	}
	db.metrics.PutCounter.Inc()
	return nil
}

// Get wraps LevelDB Get method to increment metrics counter.
func (db *DB) Get(key []byte) (value []byte, err error) {
	value, err = db.ldb.Get(key, nil)
	if err == leveldb.ErrNotFound {
		db.logger.Debugf("key %s not found during GET", string(key))
		db.metrics.GetNotFoundCounter.Inc()
		return nil, err
	} else {
		db.logger.Errorf("key %s not found in DB", string(key))
		db.metrics.GetFailCounter.Inc()
	}
	db.metrics.GetCounter.Inc()
	return value, nil
}

// Has wraps LevelDB Has method to increment metrics counter.
func (db *DB) Has(key []byte) (yes bool, err error) {
	yes, err = db.ldb.Has(key, nil)
	if err != nil {
		db.logger.Debugf("encountered error during HAS of key %s. Error: %s ", string(key), err.Error())
		db.metrics.HasFailCounter.Inc()
		return false, err
	}
	db.metrics.HasCounter.Inc()
	return yes, nil
}

// Delete wraps LevelDB Delete method to increment metrics counter.
func (db *DB) Delete(key []byte) (err error) {
	err = db.ldb.Delete(key, nil)
	if err != nil {
		db.logger.Debugf("could not DELETE key %s. Error: %s ", string(key), err.Error())
		db.metrics.DeleteFailCounter.Inc()
		return err
	}
	db.metrics.DeleteCounter.Inc()
	return nil
}

// NewIterator wraps LevelDB NewIterator method to increment metrics counter.
func (db *DB) NewIterator() iterator.Iterator {
	db.metrics.IteratorCounter.Inc()
	return db.ldb.NewIterator(nil, nil)
}

// WriteBatch wraps LevelDB Write method to increment metrics counter.
func (db *DB) WriteBatch(batch *leveldb.Batch) (err error) {
	err = db.ldb.Write(batch, nil)
	if err != nil {
		db.logger.Debugf("could not writing batch. Error: %s ", err.Error())
		db.metrics.WriteBatchFailCounter.Inc()
		return err
	}
	db.metrics.WriteBatchCounter.Inc()
	return nil
}

// Close closes LevelDB database.
func (db *DB) Close() (err error) {
	close(db.quit)
	return db.ldb.Close()
}

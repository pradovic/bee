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

package shed

import (
	"encoding/json"

	"github.com/ethersphere/bee/pkg/logging"
	"github.com/syndtr/goleveldb/leveldb"
)

// StructField is a helper to store complex structure by
// encoding it in RLP format.
type StructField struct {
	db     *DB
	key    []byte
	logger logging.Logger
}

// NewStructField returns a new StructField.
// It validates its name and type against the database schema.
func (db *DB) NewStructField(name string, logger logging.Logger) (f StructField, err error) {
	key, err := db.schemaFieldKey(name, "struct-rlp")
	if err != nil {
		return f, err
	}
	return StructField{
		db:     db,
		key:    key,
		logger: logger,
	}, nil
}

// Get unmarshals data from the database to a provided val.
// If the data is not found leveldb.ErrNotFound is returned.
func (f StructField) Get(val interface{}) (err error) {
	b, err := f.db.Get(f.key)
	if err != nil {
		f.logger.Debugf("could not GET key %s", string(f.key))
		return err
	}
	return json.Unmarshal(b, val)
}

// Put marshals provided val and saves it to the database.
func (f StructField) Put(val interface{}) (err error) {
	b, err := json.Marshal(val)
	if err != nil {
		f.logger.Debugf("could not PUT key %s", string(f.key))
		return err
	}
	return f.db.Put(f.key, b)
}

// PutInBatch marshals provided val and puts it into the batch.
func (f StructField) PutInBatch(batch *leveldb.Batch, val interface{}) (err error) {
	b, err := json.Marshal(val)
	if err != nil {
		f.logger.Debugf("could not PUT key %s in batch", string(f.key))
		return err
	}
	batch.Put(f.key, b)
	return nil
}

/*
The wrapper initially was planned to be implemented with an interface + struct style,
but Go generics are terrible even in 1.20 and do not support generics in their methods,
so this package got implemented in a C style.

Buckets(tables) should be predefined. Ideally Buckets should be referred to via Constants.

enjoy :)
*/

package boltnut

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var (
	ErrKeyNotFound = errors.New("key doesn't exist")
)

type DB struct {
	*bolt.DB
}

// Open a connection to the database.
func Init(path string, buckets *[][]byte) (*DB, error) {
	var e error
	var db DB

	db.DB, e = bolt.Open(path, 0600, nil)
	if e != nil {
		return nil, e
	}

	e = db.Update(func(tx *bolt.Tx) error {
		for idx, bucket := range *buckets {
			_, e := tx.CreateBucketIfNotExists(bucket)

			if e != nil {
				return fmt.Errorf("bucket at index %d: %s", idx, e.Error())
			}
		}
		return nil
	})

	if e != nil {
		return nil, e
	}

	return &db, nil
}

// serialise obj into a buffer.
func Serialise[T any](obj *T) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)

	e := enc.Encode((*obj))

	return &buf, e
}

// decode the byte array into obj.
func Decode[T any](obj *T, b []byte) error {
	buf := bytes.NewBuffer(b)

	dec := gob.NewDecoder(buf)

	e := dec.Decode(obj)

	return e
}

// Get a decoded value to `val` from the `bucket` by it's `key`.
func Get[T any](db DB, bucket []byte, key []byte, val *T) error {
	return db.View(func(tx *bolt.Tx) error {
		return Decode(val, tx.Bucket(bucket).Get(key))
	})
}

// Insert a new `key: val` pair to the specified bucket, or overwrite the value in case the key already exists.
func Insert[T any](db DB, bucket []byte, key []byte, val *T) error {
	return db.Update(func(tx *bolt.Tx) error {
		buf, e := Serialise(val)
		if e != nil {
			return e
		}

		return tx.Bucket(bucket).Put(key, buf.Bytes())
	})
}

// Update() first validates that `key` exists inside the `bucket`, then overwrites the value by `val`.
// If given key doesn't exist, the function returns ErrKeyNotFound without modifying the database.
// If you want to update or insert a value whenever the key exists or not, use Insert().
func Update[T any](db DB, bucket []byte, key []byte, val *T) error {
	return db.Update(func(tx *bolt.Tx) error {
		buc := tx.Bucket(bucket)

		exists := buc.Get(key)
		if exists == nil {
			return nil
		}

		buf, e := Serialise(val)
		if e != nil {
			return e
		}

		return buc.Put(key, buf.Bytes())
	})
}

// Create a new bucket. Returns an error if the bucket already exists, if the bucket name is blank, or if the bucket name is too long.
func NewBucket(db DB, bucket []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucket(bucket)
		return e
	})
}

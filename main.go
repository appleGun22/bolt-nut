/*
The wrapper initially was planned to be implemented with an interface + struct style,
but Go generics are terrible even in 1.20 and do not support generics in their methods,
so this package got partialy implemented in a C style.

Buckets(tables) should be predefined. Ideally Buckets should be referred to via Constants.
Nested Buckets are not allowed. If you need that functionallity use bbolt directly instead.

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
	bolt_db *bolt.DB
}

type Tx struct {
	bolt_tx *bolt.Tx
}

type Bucket struct {
	bolt_bucket *bolt.Bucket
}

// Open a connection to the database.
func Init(path string, buckets *[]string) (*DB, error) {
	var e error
	var db DB

	db.bolt_db, e = bolt.Open(path, 0600, nil)
	if e != nil {
		return nil, e
	}

	e = db.bolt_db.Update(func(tx *bolt.Tx) error {
		for idx, bucket := range *buckets {
			_, e := tx.CreateBucketIfNotExists([]byte(bucket))

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

// Create a new bucket. Returns an error if the bucket already exists, if the bucket name is blank, or if the bucket name is too long.
func NewBucket(db *DB, bucket string) error {
	return db.bolt_db.Update(func(tx *bolt.Tx) error {
		_, e := tx.CreateBucket([]byte(bucket))
		return e
	})
}

// Create a read only transaction. If you need to modify the database, you should use WriteTx() instead.
//
// * ReadTx() ensures nothing will be modified, and in case of an attempt to modify, an error will be returned.
//
// * Read transactions are faster for read only use cases.
func (db *DB) ReadTx(fn func(*Tx) error) error {
	return db.bolt_db.View(func(btx *bolt.Tx) error {
		return fn(NewTx(btx))
	})
}

// Create a read-write transaction. Allows to retrieve values and modify the database. If you need only to retrieve values, use ReadTx() instead.
func (db *DB) WriteTx(fn func(*Tx) error) error {
	return db.bolt_db.Update(func(btx *bolt.Tx) error {
		return fn(NewTx(btx))
	})
}

// Load the value into `val`.
func Get[T any](b *Bucket, key []byte, val *T) error {
	return Decode(val, b.bolt_bucket.Get(key))
}

// Insert a new `key: val` pair to the specified bucket, or overwrite the value in case the key already exists.
func Insert[T any](b *Bucket, key []byte, val *T) error {
	buf, e := Serialise(val)
	if e != nil {
		return e
	}

	return b.bolt_bucket.Put(key, buf.Bytes())
}

// Update() first validates that `key` exists inside the bucket, then overwrites the value by `val`.
// If given key doesn't exist, the function returns ErrKeyNotFound without modifying the database.
// If you want to update or insert a value whenever the key exists or not, use Insert().
func Update[T any](b *Bucket, key []byte, val *T) error {
	exists := b.bolt_bucket.Get(key)
	if exists == nil {
		return ErrKeyNotFound
	}

	buf, e := Serialise(val)
	if e != nil {
		return e
	}

	return b.bolt_bucket.Put(key, buf.Bytes())
}

// Execute provided function for every `key: val` pair that exist inside the bucket. Do not modify the bucket! , this will cause undefined behavior.
func ForEach(b *Bucket, fn func(k []byte, v []byte) error) error {
	return b.bolt_bucket.ForEach(fn)
}

// Access an existing bucket within a transaction. Returns nil if bucket doesn't exist.
func (tx *Tx) Bucket(name string) *Bucket {
	return &Bucket{
		bolt_bucket: tx.bolt_tx.Bucket([]byte(name)),
	}
}

func NewTx(tx *bolt.Tx) *Tx {
	return &Tx{
		bolt_tx: tx,
	}
}

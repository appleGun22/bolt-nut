/*
The wrapper initially was planned to be implemented with an interface + struct style,
but Go generics are terrible even in 1.20 and do not support generics in their methods,
so this package got partialy implemented in a C style.

Buckets(tables) should be predefined. But it's allowed to create new buckets later on.
Nested Buckets are not allowed. If you need that functionallity use bbolt directly instead.

enjoy :)
*/

package boltnut

import (
	"bytes"
	"encoding/binary"
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

type TX struct {
	bolt_tx *bolt.Tx
}

type bucket[V any] struct {
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

// Serialise obj into a buffer.
func serialise[T any](obj *T) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	enc := gob.NewEncoder(&buf)

	e := enc.Encode((*obj))

	return &buf, e
}

// Decode the byte array into obj.
func Decode[T any](obj *T, b []byte) error {
	if b == nil {
		return ErrKeyNotFound
	}

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
func (db *DB) ReadTx(fn func(*TX) error) error {
	return db.bolt_db.View(func(btx *bolt.Tx) error {
		return fn(newTx(btx))
	})
}

// Create a read-write transaction. Allows to retrieve values and modify the database. If you need only to retrieve values, use ReadTx() instead.
func (db *DB) WriteTx(fn func(*TX) error) error {
	return db.bolt_db.Update(func(btx *bolt.Tx) error {
		return fn(newTx(btx))
	})
}

// Load the value into `val`. Returns ErrKeyNotFound when given key doesn't exist.
func (b *bucket[V]) Get(key []byte, val *V) error {
	return Decode(val, b.bolt_bucket.Get(key))
}

// Insert a new `key: val` pair to the specified bucket, or overwrite the value in case the key already exists.
func (b *bucket[V]) Insert(key []byte, val *V) error {
	buf, e := serialise(val)
	if e != nil {
		return e
	}

	return b.bolt_bucket.Put(key, buf.Bytes())
}

// Delete the `key: val` pair that belongs to the provided key.
func (b *bucket[V]) Delete(key []byte) error {
	return b.bolt_bucket.Delete(key)
}

// Execute provided function for every `key: val` pair that exist inside the bucket.
//
// DO NOT MODIFY THE BUCKET while ForEach is working! this will cause undefined behavior.
func (b *bucket[V]) ForEach(fn func(k []byte, v *V) error) error {
	return b.bolt_bucket.ForEach(func(k, v []byte) error {
		var val V

		e := Decode(&val, v)
		if e != nil {
			return e
		}

		return fn(k, &val)
	})
}

// Access an existing bucket within a transaction. Returns nil if bucket doesn't exist.
func Bucket[V any](tx *TX, name string) *bucket[V] {
	return &bucket[V]{
		bolt_bucket: tx.bolt_tx.Bucket([]byte(name)),
	}
}

func newTx(tx *bolt.Tx) *TX {
	return &TX{
		bolt_tx: tx,
	}
}

// Convert any integer to bytes. Used to create keys from integers.
func Itob[INT int | int8 | int16 | int32 | uint8 | uint16 | uint32 | int64 | uint64](k INT) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(k))
	return b
}

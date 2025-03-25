package wltbase

import (
	"errors"
	"fmt"
	"io/fs"

	bolt "go.etcd.io/bbolt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DBSimpleGet retrieves a value from the BoltDB key-value store
// Returns the value associated with the given key in the specified bucket
// If the bucket or key doesn't exist, returns fs.ErrNotExist
func (e *env) DBSimpleGet(bucket, key []byte) (r []byte, err error) {
	err = e.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return fs.ErrNotExist
		}
		v := b.Get(key)
		if v == nil {
			return fs.ErrNotExist
		}
		r = make([]byte, len(v))
		copy(r, v)
		return nil
	})
	return
}

// DBSimpleDel deletes one or more keys from a bucket in the BoltDB key-value store
// If the bucket doesn't exist, the operation is considered successful
// Returns any error encountered during deletion
func (e *env) DBSimpleDel(bucket []byte, keys ...[]byte) error {
	return e.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		for _, key := range keys {
			if err := b.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// DBSimpleSet stores a key-value pair in the BoltDB key-value store
// Creates the bucket if it doesn't exist
// Returns any error encountered during the operation
func (e *env) DBSimpleSet(bucket, key, val []byte) error {
	return e.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return err
		}
		return b.Put(key, val)
	})
}

// dbDeleteBucket removes a bucket from the BoltDB key-value store
// Returns any error encountered during deletion
func (e *env) dbDeleteBucket(bucket []byte) error {
	return e.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(bucket)
	})
}

// dbSimpleIsBucketEmpty checks if a bucket in BoltDB is empty
// Returns true if the bucket doesn't exist or has no keys
// Returns false if the bucket contains at least one key
func (e *env) dbSimpleIsBucketEmpty(bucket []byte) bool {
	res := true
	e.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		k, _ := b.Cursor().First()
		if k != nil {
			// got a key → bucket is not empty
			res = false
		}
		return nil
	})
	return res
}

// FirstId retrieves the first record with the given ID and populates the result
// Translates GORM's ErrRecordNotFound to fs.ErrNotExist for consistent error handling
// Returns nil on success or any error encountered
func (e *env) FirstId(res, id any) error {
	tx := e.sql.First(res, id)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return fs.ErrNotExist
		}
		return tx.Error
	}

	return nil
}

// FirstWhere retrieves the first record matching the conditions in the where map
// Returns any error encountered during the operation
func (e *env) FirstWhere(res any, where map[string]any) error {
	tx := e.sql.Where(where).First(res)
	return tx.Error
}

// Count returns the number of records for a given model type
// Uses the provided object to determine the table
func (e *env) Count(obj any) int64 {
	var count int64
	e.sql.Model(obj).Count(&count)
	return count
}

// Delete removes a record from the database
// The object should contain a primary key value to determine what to delete
// Returns any error encountered during the operation
func (e *env) Delete(obj any) error {
	tx := e.sql.Delete(obj)
	return tx.Error
}

// AutoMigrate creates or updates the database schema based on the struct definition
// Used to ensure the database structure matches the Go structs
func (e *env) AutoMigrate(obj any) {
	e.sql.AutoMigrate(obj)
}

// DeleteAll removes all records of a specific type
// Uses a WHERE 1=1 condition to match all records
// Returns any error encountered during the operation
func (e *env) DeleteAll(obj any) error {
	tx := e.sql.Where("1 = 1").Delete(obj)
	return tx.Error
}

// DeleteWhere removes records matching the conditions in the where map
// Returns any error encountered during the operation
func (e *env) DeleteWhere(obj any, where map[string]any) error {
	tx := e.sql.Where(where).Delete(obj)
	return tx.Error
}

// Find retrieves all records matching the conditions in the where map
// Populates the target slice with the results
// Returns any error encountered during the operation
func (e *env) Find(target any, where map[string]any) error {
	tx := e.sql.Where(where).Find(target)
	return tx.Error
}

// First retrieves the first record for a model
// Populates the target with the result
// Returns any error encountered during the operation
func (e *env) First(target any) error {
	tx := e.sql.First(target)
	return tx.Error
}

// byPrimaryKey is a generic function to retrieve a record by its primary key
// Returns a pointer to the record and nil error on success
// Returns nil and fs.ErrNotExist if the record is not found
// Returns nil and any other error encountered during the operation
func byPrimaryKey[T any](e *env, id any) (*T, error) {
	var res *T
	tx := e.sql.First(&res, id)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, fs.ErrNotExist
		}
		return nil, tx.Error
	}

	return res, nil
}

// Save creates or updates a record in the database
// Uses OnConflict clause to update all fields if the record already exists
// Logs an error message if the operation fails but returns nil
// Note: This appears to have a bug as it doesn't return the error properly
func (e *env) Save(v any) error {
	res := e.sql.Clauses(clause.OnConflict{UpdateAll: true}).Create(v)
	if res.Error != nil {
		fmt.Errorf("while saving object of type %T: %w", v, res.Error)
	}
	return nil
}

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
// Returns error with context if the operation fails for other reasons
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
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("failed to get key %x from bucket %x: %w", key, bucket, err)
	}
	return
}

// DBSimpleDel deletes one or more keys from a bucket in the BoltDB key-value store
// If the bucket doesn't exist, the operation is considered successful
// Returns error with context if deletion fails
func (e *env) DBSimpleDel(bucket []byte, keys ...[]byte) error {
	err := e.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		if b == nil {
			return nil
		}
		for _, key := range keys {
			if err := b.Delete(key); err != nil {
				return fmt.Errorf("failed to delete key %x: %w", key, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete keys from bucket %x: %w", bucket, err)
	}
	return nil
}

// DBSimpleSet stores a key-value pair in the BoltDB key-value store
// Creates the bucket if it doesn't exist
// Returns error with context if the operation fails
func (e *env) DBSimpleSet(bucket, key, val []byte) error {
	err := e.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucket)
		if err != nil {
			return fmt.Errorf("failed to create bucket %x: %w", bucket, err)
		}
		return b.Put(key, val)
	})
	if err != nil {
		return fmt.Errorf("failed to set key %x in bucket %x: %w", key, bucket, err)
	}
	return nil
}

// dbDeleteBucket removes a bucket from the BoltDB key-value store
// Returns error with context if the deletion fails
func (e *env) dbDeleteBucket(bucket []byte) error {
	err := e.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(bucket)
	})
	if err != nil {
		return fmt.Errorf("failed to delete bucket %x: %w", bucket, err)
	}
	return nil
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
// Returns nil on success or error with context for other failures
func (e *env) FirstId(res, id any) error {
	tx := e.sql.First(res, id)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return fs.ErrNotExist
		}
		return fmt.Errorf("failed to find record with ID %v: %w", id, tx.Error)
	}

	return nil
}

// FirstWhere retrieves the first record matching the conditions in the where map
// Returns any error encountered during the operation with added context
func (e *env) FirstWhere(res any, where map[string]any) error {
	tx := e.sql.Where(where).First(res)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return fs.ErrNotExist
		}
		return fmt.Errorf("failed to find record with conditions %v: %w", where, tx.Error)
	}
	return nil
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
// Returns error with context if deletion fails
func (e *env) Delete(obj any) error {
	tx := e.sql.Delete(obj)
	if tx.Error != nil {
		return fmt.Errorf("failed to delete object of type %T: %w", obj, tx.Error)
	}
	return nil
}

// AutoMigrate creates or updates the database schema based on the struct definition
// Used to ensure the database structure matches the Go structs
func (e *env) AutoMigrate(obj any) {
	e.sql.AutoMigrate(obj)
}

// DeleteAll removes all records of a specific type
// Uses a WHERE 1=1 condition to match all records
// Returns error with context if deletion fails
func (e *env) DeleteAll(obj any) error {
	tx := e.sql.Where("1 = 1").Delete(obj)
	if tx.Error != nil {
		return fmt.Errorf("failed to delete all records of type %T: %w", obj, tx.Error)
	}
	return nil
}

// DeleteWhere removes records matching the conditions in the where map
// Returns error with context if deletion fails
func (e *env) DeleteWhere(obj any, where map[string]any) error {
	tx := e.sql.Where(where).Delete(obj)
	if tx.Error != nil {
		return fmt.Errorf("failed to delete records of type %T with conditions %v: %w", obj, where, tx.Error)
	}
	return nil
}

// Find retrieves all records matching the conditions in the where map
// Populates the target slice with the results
// Returns error with context if the query fails
func (e *env) Find(target any, where map[string]any) error {
	tx := e.sql.Where(where).Find(target)
	if tx.Error != nil {
		return fmt.Errorf("failed to find records with conditions %v: %w", where, tx.Error)
	}
	return nil
}

// First retrieves the first record for a model
// Populates the target with the result
// Returns fs.ErrNotExist if no record is found, or error with context for other failures
func (e *env) First(target any) error {
	tx := e.sql.First(target)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return fs.ErrNotExist
		}
		return fmt.Errorf("failed to find first record of type %T: %w", target, tx.Error)
	}
	return nil
}

// byPrimaryKey is a generic function to retrieve a record by its primary key
// Returns a pointer to the record and nil error on success
// Returns nil and fs.ErrNotExist if the record is not found
// Returns nil and error with context for other failures
func byPrimaryKey[T any](e *env, id any) (*T, error) {
	var res *T
	tx := e.sql.First(&res, id)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, fs.ErrNotExist
		}
		return nil, fmt.Errorf("failed to find record of type %T with ID %v: %w", *new(T), id, tx.Error)
	}

	return res, nil
}

// Save creates or updates a record in the database
// Uses OnConflict clause to update all fields if the record already exists
// Returns any error encountered during the save operation
func (e *env) Save(v any) error {
	res := e.sql.Clauses(clause.OnConflict{UpdateAll: true}).Create(v)
	if res.Error != nil {
		return fmt.Errorf("failed to save object of type %T: %w", v, res.Error)
	}
	return nil
}

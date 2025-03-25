package wltbase

import (
	"os"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

// TestDbSimpleIsBucketEmpty tests the improved version of dbSimpleIsBucketEmpty
func TestDbSimpleIsBucketEmpty(t *testing.T) {
	// Create a temporary environment for testing
	tempEnv, err := InitTempEnv()
	if err != nil {
		t.Fatalf("Failed to initialize temporary environment: %v", err)
	}
	defer CleanupTempEnv(tempEnv)

	// Get the environment object
	e, ok := tempEnv.(*env)
	if !ok {
		t.Fatalf("Returned environment is not a valid *env")
	}

	// Create a test bucket
	err = e.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("TestBucket"))
		return err
	})
	if err != nil {
		t.Fatalf("Failed to create bucket: %v", err)
	}

	// Test empty bucket
	isEmpty, err := e.dbSimpleIsBucketEmpty([]byte("TestBucket"))
	if err != nil {
		t.Errorf("dbSimpleIsBucketEmpty returned error for empty bucket: %v", err)
	}
	if !isEmpty {
		t.Errorf("Empty bucket should return true")
	}

	// Add a key to the bucket
	err = e.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("TestBucket"))
		return b.Put([]byte("testKey"), []byte("testValue"))
	})
	if err != nil {
		t.Fatalf("Failed to add key to bucket: %v", err)
	}

	// Test non-empty bucket
	isEmpty, err = e.dbSimpleIsBucketEmpty([]byte("TestBucket"))
	if err != nil {
		t.Errorf("dbSimpleIsBucketEmpty returned error for non-empty bucket: %v", err)
	}
	if isEmpty {
		t.Errorf("Non-empty bucket should return false")
	}

	// Test non-existent bucket
	isEmpty, err = e.dbSimpleIsBucketEmpty([]byte("NonExistentBucket"))
	if err != nil {
		t.Errorf("dbSimpleIsBucketEmpty returned error for non-existent bucket: %v", err)
	}
	if !isEmpty {
		t.Errorf("Non-existent bucket should return true (as empty)")
	}
}

// TestCountWithError tests the new CountWithError method using an in-memory SQLite database
func TestCountWithError(t *testing.T) {
	// Create a test table for our test
	type TestModel struct {
		ID   uint `gorm:"primarykey"`
		Name string
	}

	// Create a temporary environment for testing
	tempEnv, err := InitTempEnv()
	if err != nil {
		t.Fatalf("Failed to initialize temporary environment: %v", err)
	}
	defer CleanupTempEnv(tempEnv)

	// Get the environment object
	e, ok := tempEnv.(*env)
	if !ok {
		t.Fatalf("Returned environment is not a valid *env")
	}

	// Create the test table
	err = e.sql.AutoMigrate(&TestModel{})
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test with an empty table
	count, err := e.CountWithError(&TestModel{})
	if err != nil {
		t.Errorf("CountWithError returned error for empty table: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected count 0 for empty table, got %d", count)
	}

	// Add some test records
	testRecords := []TestModel{
		{Name: "Test1"},
		{Name: "Test2"},
		{Name: "Test3"},
	}
	for _, record := range testRecords {
		if err := e.sql.Create(&record).Error; err != nil {
			t.Fatalf("Failed to create test record: %v", err)
		}
	}

	// Test with populated table
	count, err = e.CountWithError(&TestModel{})
	if err != nil {
		t.Errorf("CountWithError returned error for populated table: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected count 3 for populated table, got %d", count)
	}

	// Test with error by forcing a SQL syntax error (close DB connection to force error)
	sqlDB, err := e.sql.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}
	sqlDB.Close()

	// Wait a moment to ensure the connection is fully closed
	time.Sleep(10 * time.Millisecond)

	// Now try to count again, it should fail
	count, err = e.CountWithError(&TestModel{})
	if err == nil {
		t.Errorf("Expected error for closed DB, got nil")
	}
	if count != 0 {
		t.Errorf("Expected count 0 on error, got %d", count)
	}
}

// TestInitTempEnv tests the initialization and cleanup of a temporary environment
func TestInitTempEnv(t *testing.T) {
	// Initialize a temporary environment
	tempEnv, err := InitTempEnv()
	if err != nil {
		t.Fatalf("Failed to initialize temporary environment: %v", err)
	}

	// Verify the environment was created correctly
	e, ok := tempEnv.(*env)
	if !ok {
		t.Fatalf("Returned environment is not a valid *env")
	}

	// Check if bolt DB was initialized
	if e.db == nil {
		t.Errorf("BoltDB was not initialized")
	}

	// Check if SQLite was initialized
	if e.sql == nil {
		t.Errorf("SQLite was not initialized")
	}

	// Try a simple database operation
	count, err := e.CountWithError(&currentItem{})
	if err != nil {
		t.Errorf("Failed to query database: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected empty table, got count %d", count)
	}

	// Check if temp directory exists
	if _, err := os.Stat(e.dataDir); os.IsNotExist(err) {
		t.Errorf("Temporary directory was not created: %v", err)
	}

	// Test cleanup
	err = CleanupTempEnv(tempEnv)
	if err != nil {
		t.Errorf("Failed to clean up temporary environment: %v", err)
	}

	// Verify temp directory was removed
	if _, err := os.Stat(e.dataDir); !os.IsNotExist(err) {
		t.Errorf("Temporary directory was not removed")
		// Clean up if test fails
		os.RemoveAll(e.dataDir)
	}
}

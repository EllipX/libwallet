package wltbase

import (
	"errors"
	"io/fs"
	"time"

	"gorm.io/gorm"
)

type currentItem struct {
	Key     string `gorm:"primaryKey"`
	Value   string
	Created time.Time `gorm:"autoCreateTime"`
	Updated time.Time `gorm:"autoUpdateTime"`
}

func (e *env) SetCurrent(k, v string) error {
	ci := &currentItem{Key: k}
	tx := e.sql.First(ci)
	if tx.Error != nil {
		if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return tx.Error
		}
		// not found
		ci.Value = v
		tx = e.sql.Create(ci)
		return tx.Error
	}
	// found
	ci.Value = v
	return e.Save(ci)
}

func (e *env) GetCurrent(k string) (string, error) {
	v, err := e.getCurrentItem(k)
	if err != nil {
		return "", err
	}
	return v.Value, nil
}

func (e *env) getCurrentItem(k string) (*currentItem, error) {
	ci := &currentItem{Key: k}
	tx := e.sql.First(ci)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, fs.ErrNotExist
		}
		return nil, tx.Error
	}
	return ci, nil
}

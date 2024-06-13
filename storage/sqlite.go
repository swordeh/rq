package storage

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"rq/records"
)

type SqliteRecordStore struct {
	db *gorm.DB
}

func NewSqliteRecordStore(path string) (*SqliteRecordStore, error) {

	db, err := gorm.Open(sqlite.Open(path))
	if err != nil {
		return &SqliteRecordStore{}, err
	}

	db.AutoMigrate(&records.RqRecord{})

	return &SqliteRecordStore{db: db}, nil

}

func (s *SqliteRecordStore) Add(record records.RqRecord) error {
	err := s.db.Create(&record).Error
	return err
}

func (s *SqliteRecordStore) Get(id string) (*records.RqRecord, error) {
	var record records.RqRecord
	err := s.db.Where("id = ?", id).First(&record).Error
	return &record, err
}

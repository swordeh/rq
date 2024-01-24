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

func migrateSchema() {

}

func (s *SqliteRecordStore) Add(record records.RqRecord) error {
	err := s.db.Create(record).Error
	return err
}

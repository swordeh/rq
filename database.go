package main

import (
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func OpenDatabase() error {
	switch config.Database.Engine {
	case "sqlite":
		conn, err := gorm.Open(sqlite.Open(config.Database.Filepath))
		if err != nil {
			return err
		}

		db = conn
		migrateSchema()

	default:
		return errors.New("no or unsupported database driver defined")
	}

	return nil

}

func migrateSchema() {
	db.AutoMigrate(&RqRecord{})
}

func CreateRequest(request RqRequest) {

}

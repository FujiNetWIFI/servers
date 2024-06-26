package main

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type idDB struct {
	*sqlx.DB
}

func (db *idDB) Get(dest interface{}, query string, args ...interface{}) (err error) {

	err = db.DB.Get(dest, query, args...)

	if err != sql.ErrNoRows {
		return err
	}

	return nil
}

func Must_init_db() {
	DATABASE = &idDB{DB: sqlx.MustConnect("sqlite3", "db/id.sqlite3?_foreign_keys=on&_journal=WAL&_timeout=300")}

	DB.Println("Connected to id.sqlite3")

	// https://dev.to/lefebvre/speed-up-sqlite-with-write-ahead-logging-wal-do

	// Configure Write Ahead Log
	_, err := DATABASE.Exec(`PRAGMA busy_timeout=300;PRAGMA journal_mode=WAL;PRAGMA foreign_keys=ON;`)

	if err != nil {
		DB.Fatalf("Unable to set PRAGMA correctly (%s)", err)
	}
}

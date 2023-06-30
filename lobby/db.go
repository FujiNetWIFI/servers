package main

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type lobbyDB struct {
	*sqlx.DB

	// add Errors https://pkg.go.dev/github.com/mattn/go-sqlite3@v1.14.14#ErrIoErr
}

func (db *lobbyDB) Get(dest interface{}, query string, args ...interface{}) (err error) {

	err = db.DB.Get(dest, query, args...)

	if err != sql.ErrNoRows {
		return err
	}

	return nil
}

func (db *lobbyDB) In(query string, args ...interface{}) (string, []interface{}, error) {
	return sqlx.In(query, args...)
}

func (db *lobbyDB) SelectIn(dest interface{}, query string, args ...interface{}) (err error) {

	qry, inargs, err := DATABASE.In(query, args...)
	if err != nil {
		return err
	}

	return db.DB.Select(dest, qry, inargs...)
}

func (db *lobbyDB) ExecIn(query string, args ...interface{}) (res sql.Result, err error) {

	qry, inargs, err := DATABASE.In(query, args...)
	if err != nil {
		return res, err
	}

	return db.DB.Exec(qry, inargs...)

}

func init_db() {
	DATABASE = &lobbyDB{DB: sqlx.MustConnect("sqlite3", "db/lobby.sqlite3?_foreign_keys=on&_journal=WAL&_timeout=300")}

	DB.Println("Connected to lobby.sqlite3")

	// https://dev.to/lefebvre/speed-up-sqlite-with-write-ahead-logging-wal-do

	// Configure Write Ahead Log
	_, err := DATABASE.Exec(`PRAGMA busy_timeout=300;PRAGMA journal_mode=WAL;PRAGMA foreign_keys=ON;`)

	if err != nil {
		DB.Fatalf("Unable to set PRAGMA correctly (%s)", err)
	}
}

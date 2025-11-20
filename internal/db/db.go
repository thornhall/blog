package db

import (
	"context"
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

// Creates and returns a new DB, exiting if it fails to do so.
func New() *sql.DB {
	db, err := sql.Open("sqlite", "./blog.db")
	if err != nil {
		log.Fatal(err)
	}
	initDB(db)
	return db
}

// Initializes our lightweight table post_stats, exiting on failure.
func initDB(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS post_stats (
		slug TEXT PRIMARY KEY, 
		views INTEGER DEFAULT 0, 
		likes INTEGER DEFAULT 0
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

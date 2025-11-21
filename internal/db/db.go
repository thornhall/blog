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
	db, err := sql.Open("sqlite", "file:./blog.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatal(err)
	}
	initDB(db)
	return db
}

func initDB(db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS post_stats (
		slug TEXT PRIMARY KEY, 
		views INTEGER DEFAULT 0, 
		likes INTEGER DEFAULT 0
	);`

	query2 := `CREATE TABLE IF NOT EXISTS ip_likes (
		ip TEXT,
		post_slug TEXT REFERENCES post_stats(slug),

		PRIMARY KEY (ip, post_slug)
	);`

	query3 := `CREATE TABLE IF NOT EXISTS ip_views (
		ip TEXT,
		post_slug TEXT REFERENCES post_stats(slug),

		PRIMARY KEY (ip, post_slug)
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(query2)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(query3)
	if err != nil {
		log.Fatal(err)
	}
}

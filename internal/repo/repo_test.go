package repo_test

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thornhall/blog/internal/repo"
	_ "modernc.org/sqlite"
)

var db *sql.DB

func setupTestDB() {
	var err error
	db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	query := `
    CREATE TABLE post_stats (
        slug TEXT PRIMARY KEY,
        views INTEGER DEFAULT 0,
        likes INTEGER DEFAULT 0
    );`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}
}

func TestMain(m *testing.M) {
	setupTestDB()
	code := m.Run()
	db.Close()
	os.Exit(code)
}

func TestLikes(t *testing.T) {
	txn, err := db.Begin()
	assert.NoError(t, err)
	defer txn.Rollback()

	r := repo.New(txn)
	stats, err := r.IncrementViews(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, "random-slug", stats.Slug)
	assert.Equal(t, 1, stats.Views)
	assert.Equal(t, 0, stats.Likes)

	stats, err = r.IncrementLikes(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, "random-slug", stats.Slug)
	assert.Equal(t, 1, stats.Likes)
	assert.Equal(t, 1, stats.Views)
}

func TestViews(t *testing.T) {
	txn, err := db.Begin()
	assert.NoError(t, err)
	defer txn.Rollback()

	r := repo.New(txn)
	stats, err := r.IncrementViews(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, "random-slug", stats.Slug)
	assert.Equal(t, 1, stats.Views)
	assert.Equal(t, 0, stats.Likes)

	stats, err = r.IncrementViews(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, "random-slug", stats.Slug)
	assert.Equal(t, 2, stats.Views)
	assert.Equal(t, 0, stats.Likes)
}

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

	query2 := `CREATE TABLE IF NOT EXISTS ip_likes (
		ip TEXT,
		post_slug TEXT REFERENCES post_stats(slug),

		PRIMARY KEY (ip, post_slug)
	);`

	if _, err := db.Exec(query2); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	query3 := `CREATE TABLE IF NOT EXISTS ip_views (
		ip TEXT,
		post_slug TEXT REFERENCES post_stats(slug),

		PRIMARY KEY (ip, post_slug)
	);`

	if _, err := db.Exec(query3); err != nil {
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
	r := repo.New(db)
	stats, err := r.GetStats(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, "random-slug", stats.Slug)
	assert.Equal(t, 0, stats.Views)
	assert.Equal(t, 0, stats.Likes)

	stats, err = r.IncrementLikes(t.Context(), "randomip", "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, "random-slug", stats.Slug)
	assert.Equal(t, 1, stats.Likes)
	assert.Equal(t, 0, stats.Views)
}

func TestViews(t *testing.T) {
	r := repo.New(db)
	stats, err := r.GetStats(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, "random-slug", stats.Slug)
	assert.Equal(t, 0, stats.Views)
	assert.Equal(t, 0, stats.Likes)

	stats, err = r.IncrementViews(t.Context(), "randomip", "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.Views)
}

func TestIsLiked(t *testing.T) {
	r := repo.New(db)

	stats, err := r.GetStats(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.Views)

	stats, err = r.IncrementLikes(t.Context(), "randomip2", "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.Likes)

	stats, err = r.IncrementLikes(t.Context(), "randomip2", "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.Likes)
}

func TestIsViewed(t *testing.T) {
	r := repo.New(db)

	stats, err := r.GetStats(t.Context(), "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, 0, stats.Views)

	stats, err = r.IncrementViews(t.Context(), "randomip3", "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.Views)

	stats, err = r.IncrementViews(t.Context(), "randomip3", "random-slug")
	assert.NoError(t, err)
	assert.Equal(t, 1, stats.Views)
}

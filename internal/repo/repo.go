package repo

import (
	"context"
	"github.com/thornhall/blog/internal/db"
)

type Repo struct {
	db db.DB
}

func New(db db.DB) *Repo {
	return &Repo{
		db: db,
	}
}

type Stats struct {
	Slug  string `json:"slug"`
	Likes int    `json:"likes_count"`
	Views int    `json:"views_count"`
}

func (r *Repo) IncrementViews(ctx context.Context, slug string) (Stats, error) {
	var s Stats
	s.Slug = slug

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO post_stats (slug, views, likes) VALUES (?, 1, 0)
		ON CONFLICT(slug) DO UPDATE SET views = views + 1 RETURNING views, likes
	`, slug)
	err := row.Scan(&s.Views, &s.Likes)

	return s, err
}

func (r *Repo) IncrementLikes(ctx context.Context, slug string) (Stats, error) {
	var s Stats
	s.Slug = slug

	row := r.db.QueryRowContext(ctx, "UPDATE post_stats SET likes = likes + 1 WHERE slug = ? RETURNING views, likes", slug)
	err := row.Scan(&s.Views, &s.Likes)

	return s, err
}

func (r *Repo) GetStats(ctx context.Context, slug string) (Stats, error) {
	var s Stats
	s.Slug = slug

	row := r.db.QueryRowContext(ctx, "SELECT views, likes FROM post_stats WHERE slug = ?", slug)
	err := row.Scan(&s.Views, &s.Likes)

	return s, err
}

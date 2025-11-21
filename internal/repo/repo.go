package repo

import (
	"context"
	"database/sql"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{
		db: db,
	}
}

type Stats struct {
	Slug  string `json:"slug"`
	Likes int    `json:"likes_count"`
	Views int    `json:"views_count"`
}

func (r *Repo) GetStats(ctx context.Context, slug string) (Stats, error) {
	var s Stats
	s.Slug = slug

	row := r.db.QueryRowContext(ctx, `
        INSERT INTO post_stats (slug, views, likes) 
        VALUES (?, 0, 0) 
        ON CONFLICT(slug) DO UPDATE SET views = views 
        RETURNING views, likes;
    `, slug)

	err := row.Scan(&s.Views, &s.Likes)
	return s, err
}

func (r *Repo) IncrementViews(ctx context.Context, ip, slug string) (Stats, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Stats{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
        INSERT INTO ip_views (ip, post_slug) 
        VALUES (?, ?) 
        ON CONFLICT(ip, post_slug) DO NOTHING;
    `, ip, slug)
	if err != nil {
		return Stats{}, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return Stats{}, err
	}

	if rowsAffected > 0 {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO post_stats (slug, views, likes) VALUES (?, 1, 0) 
            ON CONFLICT(slug) DO UPDATE SET views = views + 1;
        `, slug)
		if err != nil {
			return Stats{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Stats{}, err
	}

	return r.GetStats(ctx, slug)
}

func (r *Repo) IncrementLikes(ctx context.Context, ip, slug string) (Stats, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Stats{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
        INSERT INTO ip_likes (ip, post_slug) 
        VALUES (?, ?) 
        ON CONFLICT(ip, post_slug) DO NOTHING;
    `, ip, slug)
	if err != nil {
		return Stats{}, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return Stats{}, err
	}

	if rowsAffected > 0 {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO post_stats (slug, views, likes) VALUES (?, 0, 1) 
            ON CONFLICT(slug) DO UPDATE SET likes = likes + 1;
        `, slug)
		if err != nil {
			return Stats{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Stats{}, err
	}

	return r.GetStats(ctx, slug)
}

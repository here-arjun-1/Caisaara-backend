package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/community/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRepository struct {
	DB *pgxpool.Pool
}

func NewPostRepository(db *pgxpool.Pool) *PostRepository {
	return &PostRepository{
		DB: db,
	}
}

const postColumns = `
	p.id,
	p.user_id,
	p.body,
	p.created_at,
	u.username,
	pr.display_name,
	pr.avatar_url,
	(SELECT COUNT(*) FROM post_comments c WHERE c.post_id = p.id),
	(SELECT r.reaction FROM post_reactions r WHERE r.post_id = p.id AND r.user_id = $1)
FROM posts p
JOIN users u ON u.id = p.user_id
LEFT JOIN player_profiles pr ON pr.user_id = p.user_id`

func scanPost(row pgx.Row) (*model.Post, error) {
	var p model.Post

	err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.Body,
		&p.CreatedAt,
		&p.Author.Username,
		&p.Author.DisplayName,
		&p.Author.AvatarURL,
		&p.CommentCount,
		&p.MyReaction,
	)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *PostRepository) CreatePost(ctx context.Context, userID int64, body string) (int64, error) {
	var id int64

	err := r.DB.QueryRow(
		ctx,
		`INSERT INTO posts (user_id, body)
		VALUES ($1, $2)
		RETURNING id`,
		userID,
		body,
	).Scan(&id)

	return id, err
}

func (r *PostRepository) FindPostByID(ctx context.Context, viewerID, postID int64) (*model.Post, error) {
	row := r.DB.QueryRow(
		ctx,
		`SELECT `+postColumns+`
		WHERE p.id = $2`,
		viewerID,
		postID,
	)

	return scanPost(row)
}

func (r *PostRepository) ListPosts(ctx context.Context, viewerID, beforeID int64, limit int) ([]*model.Post, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT `+postColumns+`
		WHERE ($2 = 0 OR p.id < $2)
		ORDER BY p.id DESC
		LIMIT $3`,
		viewerID,
		beforeID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := []*model.Post{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, rows.Err()
}

func (r *PostRepository) FindPostOwner(ctx context.Context, postID int64) (int64, error) {
	var ownerID int64

	err := r.DB.QueryRow(
		ctx,
		`SELECT user_id FROM posts WHERE id = $1`,
		postID,
	).Scan(&ownerID)

	return ownerID, err
}

func (r *PostRepository) DeletePost(ctx context.Context, postID int64) error {
	_, err := r.DB.Exec(
		ctx,
		`DELETE FROM posts WHERE id = $1`,
		postID,
	)
	return err
}

func (r *PostRepository) CountReactions(ctx context.Context, postIDs []int64) (map[int64]map[string]int, error) {
	counts := make(map[int64]map[string]int)
	if len(postIDs) == 0 {
		return counts, nil
	}

	rows, err := r.DB.Query(
		ctx,
		`SELECT post_id, reaction, COUNT(*)
		FROM post_reactions
		WHERE post_id = ANY($1)
		GROUP BY post_id, reaction`,
		postIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var postID int64
		var reaction string
		var count int

		if err := rows.Scan(&postID, &reaction, &count); err != nil {
			return nil, err
		}
		if counts[postID] == nil {
			counts[postID] = make(map[string]int)
		}
		counts[postID][reaction] = count
	}

	return counts, rows.Err()
}

func (r *PostRepository) SetReaction(ctx context.Context, postID, userID int64, reaction string) error {
	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO post_reactions (post_id, user_id, reaction)
		VALUES ($1, $2, $3)
		ON CONFLICT (post_id, user_id) DO UPDATE SET
			reaction   = EXCLUDED.reaction,
			created_at = now()`,
		postID,
		userID,
		reaction,
	)
	return err
}

func (r *PostRepository) DeleteReaction(ctx context.Context, postID, userID int64) error {
	_, err := r.DB.Exec(
		ctx,
		`DELETE FROM post_reactions WHERE post_id = $1 AND user_id = $2`,
		postID,
		userID,
	)
	return err
}

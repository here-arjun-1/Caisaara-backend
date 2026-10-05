package repository

import (
	"context"

	"github.com/here-arjun-1/Caisaara-backend/internal/community/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	DB *pgxpool.Pool
}

func NewCommentRepository(db *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{
		DB: db,
	}
}

func (r *CommentRepository) CreateComment(ctx context.Context, postID, userID int64, body string) (*model.Comment, error) {
	var c model.Comment

	err := r.DB.QueryRow(
		ctx,
		`WITH inserted AS (
			INSERT INTO post_comments (post_id, user_id, body)
			VALUES ($1, $2, $3)
			RETURNING id, post_id, user_id, body, created_at
		)
		SELECT i.id, i.post_id, i.user_id, i.body, i.created_at,
			u.username, pr.display_name, pr.avatar_url
		FROM inserted i
		JOIN users u ON u.id = i.user_id
		LEFT JOIN player_profiles pr ON pr.user_id = i.user_id`,
		postID,
		userID,
		body,
	).Scan(
		&c.ID,
		&c.PostID,
		&c.UserID,
		&c.Body,
		&c.CreatedAt,
		&c.Author.Username,
		&c.Author.DisplayName,
		&c.Author.AvatarURL,
	)

	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *CommentRepository) ListComments(ctx context.Context, postID, afterID int64, limit int) ([]*model.Comment, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT c.id, c.post_id, c.user_id, c.body, c.created_at,
			u.username, pr.display_name, pr.avatar_url
		FROM post_comments c
		JOIN users u ON u.id = c.user_id
		LEFT JOIN player_profiles pr ON pr.user_id = c.user_id
		WHERE c.post_id = $1 AND c.id > $2
		ORDER BY c.id
		LIMIT $3`,
		postID,
		afterID,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []*model.Comment{}
	for rows.Next() {
		var c model.Comment

		err := rows.Scan(
			&c.ID,
			&c.PostID,
			&c.UserID,
			&c.Body,
			&c.CreatedAt,
			&c.Author.Username,
			&c.Author.DisplayName,
			&c.Author.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}

	return comments, rows.Err()
}

func (r *CommentRepository) FindCommentOwners(ctx context.Context, commentID int64) (int64, int64, error) {
	var commentOwnerID, postOwnerID int64

	err := r.DB.QueryRow(
		ctx,
		`SELECT c.user_id, p.user_id
		FROM post_comments c
		JOIN posts p ON p.id = c.post_id
		WHERE c.id = $1`,
		commentID,
	).Scan(&commentOwnerID, &postOwnerID)

	return commentOwnerID, postOwnerID, err
}

func (r *CommentRepository) DeleteComment(ctx context.Context, commentID int64) error {
	_, err := r.DB.Exec(
		ctx,
		`DELETE FROM post_comments WHERE id = $1`,
		commentID,
	)
	return err
}

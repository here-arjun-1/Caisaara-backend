package club

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClubRepository interface {
	CreateClub(ctx context.Context, club *Club) error
	FindClubByID(ctx context.Context, clubID string) (*Club, error)
	ListClubs(ctx context.Context, limit int, offset int) ([]Club, error)
	AddMember(ctx context.Context, clubID string, userID int64) error
	RemoveMember(ctx context.Context, clubID string, userID int64) error
	GetMemberRole(ctx context.Context, clubID string, userID int64) (string, error)
	ListMembers(ctx context.Context, clubID string) ([]Member, error)
	CreateMessage(ctx context.Context, msg *Message) error
	GetRecentMessages(ctx context.Context, clubID string, limit int) ([]Message, error)
	GetLeaderboard(ctx context.Context, clubID string) ([]LeaderboardEntry, error)
}

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) ClubRepository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateClub(ctx context.Context, club *Club) error {
	club.ID = uuid.New().String()

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin create club tx failed", "error", err)
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	err = tx.QueryRow(
		ctx,
		`INSERT INTO clubs (id, name, description, owner_id)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`,
		club.ID,
		club.Name,
		club.Description,
		club.OwnerID,
	).Scan(&club.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrClubNameTaken
		}

		slog.ErrorContext(ctx, "create club failed", "owner_id", club.OwnerID, "error", err)
		return err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO club_members (club_id, user_id, role) VALUES ($1, $2, $3)`,
		club.ID,
		club.OwnerID,
		RoleOwner,
	)
	if err != nil {
		slog.ErrorContext(ctx, "add owner to club failed", "club_id", club.ID, "error", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit create club tx failed", "club_id", club.ID, "error", err)
		return err
	}

	club.MemberCount = 1
	return nil
}

func (r *Repository) FindClubByID(ctx context.Context, clubID string) (*Club, error) {
	if _, err := uuid.Parse(clubID); err != nil {
		return nil, ErrClubNotFound
	}

	var club Club
	err := r.DB.QueryRow(
		ctx,
		`SELECT
			c.id,
			c.name,
			c.description,
			c.owner_id,
			(SELECT COUNT(*) FROM club_members WHERE club_id = c.id) AS member_count,
			c.created_at
		FROM clubs c
		WHERE c.id = $1`,
		clubID,
	).Scan(
		&club.ID,
		&club.Name,
		&club.Description,
		&club.OwnerID,
		&club.MemberCount,
		&club.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClubNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "find club failed", "club_id", clubID, "error", err)
		return nil, err
	}

	return &club, nil
}

func (r *Repository) ListClubs(ctx context.Context, limit int, offset int) ([]Club, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			c.id,
			c.name,
			c.description,
			c.owner_id,
			(SELECT COUNT(*) FROM club_members WHERE club_id = c.id) AS member_count,
			c.created_at
		FROM clubs c
		ORDER BY c.created_at DESC
		LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "list clubs failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	clubs := []Club{}
	for rows.Next() {
		var club Club
		err := rows.Scan(
			&club.ID,
			&club.Name,
			&club.Description,
			&club.OwnerID,
			&club.MemberCount,
			&club.CreatedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan club failed", "error", err)
			return nil, err
		}
		clubs = append(clubs, club)
	}

	return clubs, rows.Err()
}

func (r *Repository) AddMember(ctx context.Context, clubID string, userID int64) error {
	_, err := r.DB.Exec(
		ctx,
		`INSERT INTO club_members (club_id, user_id, role) VALUES ($1, $2, $3)`,
		clubID,
		userID,
		RoleMember,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyMember
		}

		slog.ErrorContext(ctx, "add club member failed", "club_id", clubID, "user_id", userID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) RemoveMember(ctx context.Context, clubID string, userID int64) error {
	tag, err := r.DB.Exec(
		ctx,
		`DELETE FROM club_members WHERE club_id = $1 AND user_id = $2`,
		clubID,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "remove club member failed", "club_id", clubID, "user_id", userID, "error", err)
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}

	return nil
}

func (r *Repository) GetMemberRole(ctx context.Context, clubID string, userID int64) (string, error) {
	var role string
	err := r.DB.QueryRow(
		ctx,
		`SELECT role FROM club_members WHERE club_id = $1 AND user_id = $2`,
		clubID,
		userID,
	).Scan(&role)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotMember
	}
	if err != nil {
		slog.ErrorContext(ctx, "get member role failed", "club_id", clubID, "user_id", userID, "error", err)
		return "", err
	}

	return role, nil
}

func (r *Repository) ListMembers(ctx context.Context, clubID string) ([]Member, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT m.user_id, u.username, u.rating, m.role, m.joined_at
		FROM club_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.club_id = $1
		ORDER BY m.joined_at`,
		clubID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "list club members failed", "club_id", clubID, "error", err)
		return nil, err
	}
	defer rows.Close()

	members := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.UserID, &m.Username, &m.Rating, &m.Role, &m.JoinedAt); err != nil {
			slog.ErrorContext(ctx, "scan club member failed", "club_id", clubID, "error", err)
			return nil, err
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

func (r *Repository) CreateMessage(ctx context.Context, msg *Message) error {
	msg.ID = uuid.New().String()

	err := r.DB.QueryRow(
		ctx,
		`INSERT INTO club_messages (id, club_id, user_id, message)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, (SELECT username FROM users WHERE id = $3)`,
		msg.ID,
		msg.ClubID,
		msg.UserID,
		msg.Message,
	).Scan(&msg.CreatedAt, &msg.Username)

	if err != nil {
		slog.ErrorContext(ctx, "create club message failed", "club_id", msg.ClubID, "user_id", msg.UserID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) GetRecentMessages(ctx context.Context, clubID string, limit int) ([]Message, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT id, club_id, user_id, username, message, created_at
		FROM (
			SELECT cm.id, cm.club_id, cm.user_id, u.username, cm.message, cm.created_at
			FROM club_messages cm
			JOIN users u ON u.id = cm.user_id
			WHERE cm.club_id = $1
			ORDER BY cm.created_at DESC
			LIMIT $2
		) recent
		ORDER BY created_at`,
		clubID,
		limit,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get club messages failed", "club_id", clubID, "error", err)
		return nil, err
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.ClubID, &msg.UserID, &msg.Username, &msg.Message, &msg.CreatedAt); err != nil {
			slog.ErrorContext(ctx, "scan club message failed", "club_id", clubID, "error", err)
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (r *Repository) GetLeaderboard(ctx context.Context, clubID string) ([]LeaderboardEntry, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			RANK() OVER (ORDER BY u.rating DESC) AS rank,
			u.id,
			u.username,
			u.rating
		FROM club_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.club_id = $1
		ORDER BY u.rating DESC, u.username`,
		clubID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "get club leaderboard failed", "club_id", clubID, "error", err)
		return nil, err
	}
	defer rows.Close()

	entries := []LeaderboardEntry{}
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.Rank, &e.UserID, &e.Username, &e.Rating); err != nil {
			slog.ErrorContext(ctx, "scan leaderboard entry failed", "club_id", clubID, "error", err)
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

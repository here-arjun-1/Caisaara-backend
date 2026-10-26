package club

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClubRepository interface {
	CreateClub(ctx context.Context, club *Club) error
	FindClubByID(ctx context.Context, clubID string) (*Club, error)
	ListClubs(ctx context.Context, search string, limit int, offset int) ([]Club, error)
	AddMember(ctx context.Context, clubID string, userID int64) error
	RemoveMember(ctx context.Context, clubID string, userID int64) error
	GetMemberRole(ctx context.Context, clubID string, userID int64) (string, error)
	ListMembers(ctx context.Context, clubID string) ([]Member, error)
	CreateMessage(ctx context.Context, msg *Message) error
	GetRecentMessages(ctx context.Context, clubID string, limit int) ([]Message, error)
	GetLeaderboard(ctx context.Context, clubID string, mode string) ([]LeaderboardEntry, error)
	UpdateMemberRole(ctx context.Context, clubID string, userID int64, role string) error
	TransferOwnership(ctx context.Context, clubID string, oldOwnerID int64, newOwnerID int64) error
	BlockUser(ctx context.Context, clubID string, userID int64, blockedBy int64, blockedUntil *time.Time) error
	IsBlocked(ctx context.Context, clubID string, userID int64) (bool, *time.Time, error)
	RemoveBlock(ctx context.Context, clubID string, userID int64) error
	ListBlocked(ctx context.Context, clubID string) ([]BlockedUser, error)
	ListMyClubs(ctx context.Context, userID int64) ([]MyClub, error)
	FindClubByShareCode(ctx context.Context, shareCode string) (*Club, error)
	UpdateShareCode(ctx context.Context, clubID string, newCode string, expiresAt time.Time) error
	DeleteClub(ctx context.Context, clubID string) error
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
		`INSERT INTO clubs (id, name, description, owner_id, share_code, share_code_expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`,
		club.ID,
		club.Name,
		club.Description,
		club.OwnerID,
		club.ShareCode,
		club.ShareCodeExpiresAt,
	).Scan(&club.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "clubs_name_key" {
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
			c.share_code,
			c.share_code_expires_at,
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
		&club.ShareCode,
		&club.ShareCodeExpiresAt,
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

	if club.ShareCode != "" {
		club.Link = "https://caisaara.duckdns.org/club/" + club.ShareCode
	}

	return &club, nil
}

func (r *Repository) ListClubs(ctx context.Context, search string, limit int, offset int) ([]Club, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			c.id,
			c.name,
			c.description,
			c.owner_id,
			c.share_code,
			c.share_code_expires_at,
			(SELECT COUNT(*) FROM club_members WHERE club_id = c.id) AS member_count,
			c.created_at
		FROM clubs c
		WHERE $1::text = '' OR c.name ILIKE '%' || $1 || '%'
		ORDER BY c.created_at DESC
		LIMIT $2 OFFSET $3`,
		search,
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
			&club.ShareCode,
			&club.ShareCodeExpiresAt,
			&club.MemberCount,
			&club.CreatedAt,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan club failed", "error", err)
			return nil, err
		}
		if club.ShareCode != "" {
			club.Link = "https://caisaara.duckdns.org/club/" + club.ShareCode
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

func (r *Repository) GetLeaderboard(ctx context.Context, clubID string, mode string) ([]LeaderboardEntry, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			RANK() OVER (ORDER BY COALESCE(pr.rating, u.start_rating) DESC) AS rank,
			u.id,
			u.username,
			COALESCE(pr.rating, u.start_rating) AS rating
		FROM club_members m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN player_ratings pr ON pr.user_id = m.user_id AND pr.mode = $2
		WHERE m.club_id = $1
		ORDER BY rating DESC, u.username`,
		clubID,
		mode,
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

func (r *Repository) UpdateMemberRole(ctx context.Context, clubID string, userID int64, role string) error {
	tag, err := r.DB.Exec(
		ctx,
		`UPDATE club_members SET role = $3 WHERE club_id = $1 AND user_id = $2`,
		clubID,
		userID,
		role,
	)

	if err != nil {
		slog.ErrorContext(ctx, "update member role failed", "club_id", clubID, "user_id", userID, "error", err)
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotMember
	}

	return nil
}

func (r *Repository) TransferOwnership(ctx context.Context, clubID string, oldOwnerID int64, newOwnerID int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin transfer ownership tx failed", "error", err)
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(
		ctx,
		`UPDATE club_members SET role = $3 WHERE club_id = $1 AND user_id = $2 AND role = $4`,
		clubID,
		oldOwnerID,
		RoleAdmin,
		RoleOwner,
	)
	if err != nil {
		slog.ErrorContext(ctx, "demote old owner failed", "club_id", clubID, "user_id", oldOwnerID, "error", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotAllowed
	}

	tag, err = tx.Exec(
		ctx,
		`UPDATE club_members SET role = $3 WHERE club_id = $1 AND user_id = $2 AND role = $4`,
		clubID,
		newOwnerID,
		RoleOwner,
		RoleAdmin,
	)
	if err != nil {
		slog.ErrorContext(ctx, "promote new owner failed", "club_id", clubID, "user_id", newOwnerID, "error", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrTargetNotAdmin
	}

	_, err = tx.Exec(ctx, `UPDATE clubs SET owner_id = $2 WHERE id = $1`, clubID, newOwnerID)
	if err != nil {
		slog.ErrorContext(ctx, "update club owner_id failed", "club_id", clubID, "error", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit transfer ownership tx failed", "club_id", clubID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) BlockUser(ctx context.Context, clubID string, userID int64, blockedBy int64, blockedUntil *time.Time) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin block user tx failed", "error", err)
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `DELETE FROM club_members WHERE club_id = $1 AND user_id = $2`, clubID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "remove member before block failed", "club_id", clubID, "user_id", userID, "error", err)
		return err
	}

	_, err = tx.Exec(
		ctx,
		`INSERT INTO club_bans (club_id, user_id, banned_by, banned_until)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (club_id, user_id)
		DO UPDATE SET banned_by = EXCLUDED.banned_by, banned_until = EXCLUDED.banned_until, created_at = NOW()`,
		clubID,
		userID,
		blockedBy,
		blockedUntil,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrUserNotFound
		}
		slog.ErrorContext(ctx, "save club ban failed", "club_id", clubID, "user_id", userID, "error", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit block user tx failed", "club_id", clubID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) IsBlocked(ctx context.Context, clubID string, userID int64) (bool, *time.Time, error) {
	var blockedUntil *time.Time
	err := r.DB.QueryRow(
		ctx,
		`SELECT banned_until FROM club_bans
		WHERE club_id = $1 AND user_id = $2
		AND (banned_until IS NULL OR banned_until > NOW())`,
		clubID,
		userID,
	).Scan(&blockedUntil)

	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "check club ban failed", "club_id", clubID, "user_id", userID, "error", err)
		return false, nil, err
	}

	return true, blockedUntil, nil
}

func (r *Repository) RemoveBlock(ctx context.Context, clubID string, userID int64) error {
	tag, err := r.DB.Exec(ctx, `DELETE FROM club_bans WHERE club_id = $1 AND user_id = $2`, clubID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "remove club ban failed", "club_id", clubID, "user_id", userID, "error", err)
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotBlocked
	}

	return nil
}

func (r *Repository) ListBlocked(ctx context.Context, clubID string) ([]BlockedUser, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT b.user_id, u.username, b.banned_by, b.banned_until, b.created_at
		FROM club_bans b
		JOIN users u ON u.id = b.user_id
		WHERE b.club_id = $1
		AND (b.banned_until IS NULL OR b.banned_until > NOW())
		ORDER BY b.created_at DESC`,
		clubID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "list club bans failed", "club_id", clubID, "error", err)
		return nil, err
	}
	defer rows.Close()

	blocked := []BlockedUser{}
	for rows.Next() {
		var b BlockedUser
		if err := rows.Scan(&b.UserID, &b.Username, &b.BlockedBy, &b.BlockedUntil, &b.CreatedAt); err != nil {
			slog.ErrorContext(ctx, "scan club ban failed", "club_id", clubID, "error", err)
			return nil, err
		}
		blocked = append(blocked, b)
	}

	return blocked, rows.Err()
}

func (r *Repository) ListMyClubs(ctx context.Context, userID int64) ([]MyClub, error) {
	rows, err := r.DB.Query(
		ctx,
		`SELECT
			c.id,
			c.name,
			c.description,
			c.owner_id,
			c.share_code,
			c.share_code_expires_at,
			(SELECT COUNT(*) FROM club_members WHERE club_id = c.id) AS member_count,
			c.created_at,
			m.role
		FROM club_members m
		JOIN clubs c ON c.id = m.club_id
		WHERE m.user_id = $1
		ORDER BY m.joined_at DESC`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "list my clubs failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	clubs := []MyClub{}
	for rows.Next() {
		var club MyClub
		err := rows.Scan(
			&club.ID,
			&club.Name,
			&club.Description,
			&club.OwnerID,
			&club.ShareCode,
			&club.ShareCodeExpiresAt,
			&club.MemberCount,
			&club.CreatedAt,
			&club.Role,
		)
		if err != nil {
			slog.ErrorContext(ctx, "scan my club failed", "user_id", userID, "error", err)
			return nil, err
		}
		if club.ShareCode != "" {
			club.Link = "https://caisaara.duckdns.org/club/" + club.ShareCode
		}
		clubs = append(clubs, club)
	}

	return clubs, rows.Err()
}

func (r *Repository) FindClubByShareCode(ctx context.Context, shareCode string) (*Club, error) {
	var club Club
	err := r.DB.QueryRow(
		ctx,
		`SELECT
			c.id,
			c.name,
			c.description,
			c.owner_id,
			c.share_code,
			c.share_code_expires_at,
			(SELECT COUNT(*) FROM club_members WHERE club_id = c.id) AS member_count,
			c.created_at
		FROM clubs c
		WHERE c.share_code = $1`,
		shareCode,
	).Scan(
		&club.ID,
		&club.Name,
		&club.Description,
		&club.OwnerID,
		&club.ShareCode,
		&club.ShareCodeExpiresAt,
		&club.MemberCount,
		&club.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrClubNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "find club by share code failed", "error", err)
		return nil, err
	}

	if club.ShareCodeExpiresAt != nil && time.Now().After(*club.ShareCodeExpiresAt) {
		return nil, ErrShareCodeExpired
	}

	if club.ShareCode != "" {
		club.Link = "https://caisaara.duckdns.org/club/" + club.ShareCode
	}

	return &club, nil
}

func (r *Repository) UpdateShareCode(ctx context.Context, clubID string, newCode string, expiresAt time.Time) error {
	_, err := r.DB.Exec(
		ctx,
		`UPDATE clubs SET share_code = $1, share_code_expires_at = $2 WHERE id = $3`,
		newCode,
		expiresAt,
		clubID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update share code failed", "club_id", clubID, "error", err)
		return err
	}
	return nil
}

func (r *Repository) DeleteClub(ctx context.Context, clubID string) error {
	tag, err := r.DB.Exec(ctx, `DELETE FROM clubs WHERE id = $1`, clubID)
	if err != nil {
		slog.ErrorContext(ctx, "delete club failed", "club_id", clubID, "error", err)
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrClubNotFound
	}

	return nil
}

package admin

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Repository struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func NewRepository(db *pgxpool.Pool, redisClient *redis.Client) *Repository {
	return &Repository{
		DB:    db,
		Redis: redisClient,
	}
}

func logAction(ctx context.Context, tx pgx.Tx, adminID int64, action string, targetType string, targetID string, details string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO admin_actions (admin_id, action, target_type, target_id, details)
		VALUES ($1, $2, $3, $4, $5)`,
		adminID,
		action,
		targetType,
		targetID,
		details,
	)
	return err
}

func (r *Repository) LogAction(ctx context.Context, adminID int64, action string, targetType string, targetID string, details string) error {
	_, err := r.DB.Exec(ctx,
		`INSERT INTO admin_actions (admin_id, action, target_type, target_id, details)
		VALUES ($1, $2, $3, $4, $5)`,
		adminID,
		action,
		targetType,
		targetID,
		details,
	)
	return err
}

func (r *Repository) GetStats(ctx context.Context) (*Stats, error) {
	var stats Stats
	err := r.DB.QueryRow(ctx,
		`SELECT
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND role = 'user'),
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND role = 'user' AND created_at >= CURRENT_DATE),
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND role = 'user' AND banned_until > NOW()),
			(SELECT COUNT(*) FROM games WHERE status = 'active'),
			(SELECT COUNT(*) FROM player_reports WHERE status = 'pending'),
			(SELECT COUNT(*) FROM tournaments WHERE status = 'ongoing')`,
	).Scan(
		&stats.TotalUsers,
		&stats.NewUsersToday,
		&stats.BannedUsers,
		&stats.ActiveGames,
		&stats.PendingReports,
		&stats.OngoingTournaments,
	)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *Repository) SearchUsers(ctx context.Context, search string, limit int, offset int) ([]User, error) {
	pattern := "%" + search + "%"

	rows, err := r.DB.Query(ctx,
		`SELECT
			id,
			username,
			COALESCE(email, ''),
			role,
			rating,
			COALESCE(banned_until > NOW(), false),
			ban_reason,
			created_at
		FROM users
		WHERE deleted_at IS NULL
			AND (username ILIKE $1 OR COALESCE(email, '') ILIKE $1)
		ORDER BY id DESC
		LIMIT $2 OFFSET $3`,
		pattern,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
			&user.Rating,
			&user.IsBanned,
			&user.BanReason,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *Repository) GetUser(ctx context.Context, userID int64) (*UserDetail, error) {
	var user UserDetail
	err := r.DB.QueryRow(ctx,
		`SELECT
			u.id,
			u.username,
			COALESCE(u.email, ''),
			u.role,
			u.rating,
			COALESCE(u.email_verified, false),
			COALESCE(u.banned_until > NOW(), false),
			COALESCE(u.banned_until = 'infinity', false),
			CASE WHEN u.banned_until > NOW() AND u.banned_until <> 'infinity' THEN u.banned_until END,
			u.ban_reason,
			(SELECT COUNT(*) FROM games g WHERE (g.white_player_id = u.id OR g.black_player_id = u.id) AND g.status = 'finished'),
			(SELECT COUNT(*) FROM player_reports pr WHERE pr.reported_id = u.id),
			u.created_at
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL`,
		userID,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
		&user.Rating,
		&user.EmailVerified,
		&user.IsBanned,
		&user.IsPermanentBan,
		&user.BannedUntil,
		&user.BanReason,
		&user.GamesPlayed,
		&user.ReportsAgainst,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetUserRole(ctx context.Context, userID int64) (string, error) {
	var role string
	err := r.DB.QueryRow(ctx,
		`SELECT role FROM users WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	return role, nil
}

func (r *Repository) BanUser(ctx context.Context, adminID int64, userID int64, days int, reason string, details string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(ctx,
		`UPDATE users
		SET banned_until = CASE WHEN $2 = 0 THEN 'infinity'::timestamptz ELSE NOW() + make_interval(days => $2) END,
			ban_reason = $3,
			password_version = COALESCE(password_version, 1) + 1
		WHERE id = $1 AND deleted_at IS NULL`,
		userID,
		days,
		reason,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	_, err = tx.Exec(ctx,
		`UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return err
	}

	err = logAction(ctx, tx, adminID, "ban_user", "user", strconv.FormatInt(userID, 10), details)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) UnbanUser(ctx context.Context, adminID int64, userID int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(ctx,
		`UPDATE users
		SET banned_until = NULL, ban_reason = ''
		WHERE id = $1 AND deleted_at IS NULL AND banned_until > NOW()`,
		userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotBanned
	}

	err = logAction(ctx, tx, adminID, "unban_user", "user", strconv.FormatInt(userID, 10), "")
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) ForceLogout(ctx context.Context, adminID int64, userID int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	tag, err := tx.Exec(ctx,
		`UPDATE users SET password_version = COALESCE(password_version, 1) + 1 WHERE id = $1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	_, err = tx.Exec(ctx,
		`UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return err
	}

	err = logAction(ctx, tx, adminID, "force_logout", "user", strconv.FormatInt(userID, 10), "")
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) ListReports(ctx context.Context, status string, limit int, offset int) ([]Report, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT
			pr.id,
			pr.reporter_id,
			reporter.username,
			pr.reported_id,
			reported.username,
			pr.reason,
			pr.details,
			pr.status,
			pr.created_at
		FROM player_reports pr
		JOIN users reporter ON reporter.id = pr.reporter_id
		JOIN users reported ON reported.id = pr.reported_id
		WHERE pr.status = $1
		ORDER BY pr.created_at DESC
		LIMIT $2 OFFSET $3`,
		status,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := []Report{}
	for rows.Next() {
		var report Report
		err := rows.Scan(
			&report.ID,
			&report.ReporterID,
			&report.ReporterUsername,
			&report.ReportedID,
			&report.ReportedUsername,
			&report.Reason,
			&report.Details,
			&report.Status,
			&report.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *Repository) CloseReport(ctx context.Context, adminID int64, reportID int64, status string, note string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var currentStatus string
	err = tx.QueryRow(ctx,
		`SELECT status FROM player_reports WHERE id = $1 FOR UPDATE`,
		reportID,
	).Scan(&currentStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrReportNotFound
	}
	if err != nil {
		return err
	}

	if currentStatus != ReportPending {
		return ErrReportAlreadyClosed
	}

	_, err = tx.Exec(ctx,
		`UPDATE player_reports SET status = $2 WHERE id = $1`,
		reportID,
		status,
	)
	if err != nil {
		return err
	}

	action := "resolve_report"
	if status == ReportDismissed {
		action = "dismiss_report"
	}

	err = logAction(ctx, tx, adminID, action, "report", strconv.FormatInt(reportID, 10), note)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) DeletePost(ctx context.Context, adminID int64, postID int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var authorID int64
	err = tx.QueryRow(ctx,
		`DELETE FROM posts WHERE id = $1 RETURNING user_id`,
		postID,
	).Scan(&authorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPostNotFound
	}
	if err != nil {
		return err
	}

	err = logAction(ctx, tx, adminID, "delete_post", "post", strconv.FormatInt(postID, 10), fmt.Sprintf("author %d", authorID))
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) DeleteComment(ctx context.Context, adminID int64, commentID int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var authorID int64
	var postID int64
	err = tx.QueryRow(ctx,
		`DELETE FROM post_comments WHERE id = $1 RETURNING user_id, post_id`,
		commentID,
	).Scan(&authorID, &postID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCommentNotFound
	}
	if err != nil {
		return err
	}

	err = logAction(ctx, tx, adminID, "delete_comment", "comment", strconv.FormatInt(commentID, 10), fmt.Sprintf("author %d on post %d", authorID, postID))
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) DeleteClub(ctx context.Context, adminID int64, clubID string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var name string
	var ownerID int64
	err = tx.QueryRow(ctx,
		`DELETE FROM clubs WHERE id = $1 RETURNING name, owner_id`,
		clubID,
	).Scan(&name, &ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrClubNotFound
	}
	if err != nil {
		return err
	}

	err = logAction(ctx, tx, adminID, "delete_club", "club", clubID, fmt.Sprintf("%s (owner %d)", name, ownerID))
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) ListActions(ctx context.Context, limit int, offset int) ([]Action, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT
			a.id,
			a.admin_id,
			COALESCE(u.username, ''),
			a.action,
			a.target_type,
			a.target_id,
			a.details,
			a.created_at
		FROM admin_actions a
		LEFT JOIN users u ON u.id = a.admin_id
		ORDER BY a.id DESC
		LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []Action{}
	for rows.Next() {
		var action Action
		err := rows.Scan(
			&action.ID,
			&action.AdminID,
			&action.AdminUsername,
			&action.Action,
			&action.TargetType,
			&action.TargetID,
			&action.Details,
			&action.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return actions, nil
}

func (r *Repository) ListLiveGames(ctx context.Context, limit int, offset int) ([]LiveGame, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT
			g.id::text,
			g.white_player_id,
			COALESCE(wu.username, ''),
			g.black_player_id,
			COALESCE(bu.username, ''),
			COALESCE(g.time_control_mode, ''),
			g.time_control_minutes,
			g.rated,
			(SELECT COUNT(*) FROM game_moves m WHERE m.game_id = g.id),
			g.created_at,
			g.started_at
		FROM games g
		LEFT JOIN users wu ON wu.id = g.white_player_id
		LEFT JOIN users bu ON bu.id = g.black_player_id
		WHERE g.status = 'active'
		ORDER BY g.created_at ASC
		LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := []LiveGame{}
	for rows.Next() {
		var g LiveGame
		err := rows.Scan(
			&g.ID,
			&g.WhitePlayerID,
			&g.WhiteUsername,
			&g.BlackPlayerID,
			&g.BlackUsername,
			&g.TimeControlMode,
			&g.TimeControlMinutes,
			&g.Rated,
			&g.MoveCount,
			&g.CreatedAt,
			&g.StartedAt,
		)
		if err != nil {
			return nil, err
		}
		games = append(games, g)
	}

	return games, rows.Err()
}

func (r *Repository) ListMatchmakingQueues(ctx context.Context) ([]MatchmakingQueue, error) {
	queues := []MatchmakingQueue{}

	iter := r.Redis.Scan(ctx, 0, "matchmaking:queue:*", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()

		count, err := r.Redis.ZCard(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		queues = append(queues, MatchmakingQueue{
			Queue:   strings.TrimPrefix(key, "matchmaking:queue:"),
			Players: count,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return queues, nil
}

func (r *Repository) CheckHealth(ctx context.Context) Health {
	health := Health{
		Database: HealthOK,
		Redis:    HealthOK,
	}

	if err := r.DB.Ping(ctx); err != nil {
		health.Database = HealthDown
	}

	if err := r.Redis.Ping(ctx).Err(); err != nil {
		health.Redis = HealthDown
	}

	return health
}

func (r *Repository) IsTournamentGame(ctx context.Context, gameID string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM tournament_pairings WHERE game_id = $1)`,
		gameID,
	).Scan(&exists)
	return exists, err
}

package friends

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		DB: db,
	}
}

func (r *Repository) CreateRequest(ctx context.Context, senderID int64, receiverID int64) (int64, error) {
	var requestID int64
	err := r.DB.QueryRow(ctx,
		`INSERT INTO friend_requests (sender_id, receiver_id) VALUES ($1, $2) RETURNING id`,
		senderID, receiverID,
	).Scan(&requestID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrRequestExists
		}

		slog.ErrorContext(ctx, "create friend request failed", "sender_id", senderID, "receiver_id", receiverID, "error", err)
		return 0, err
	}
	return requestID, nil
}

func (r *Repository) FindPendingRequests(ctx context.Context, userID int64) ([]FriendRequest, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT fr.id, u.id, u.username, u.rating, fr.created_at
		FROM friend_requests fr
		JOIN users u ON u.id = fr.sender_id
		WHERE fr.receiver_id = $1 AND fr.status = $2
		ORDER BY fr.created_at DESC`,
		userID, StatusPending,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find pending friend requests failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	requests := []FriendRequest{}
	for rows.Next() {
		var request FriendRequest
		err := rows.Scan(&request.ID, &request.SenderID, &request.SenderUsername, &request.SenderRating, &request.CreatedAt)
		if err != nil {
			slog.ErrorContext(ctx, "scan friend request row failed", "user_id", userID, "error", err)
			return nil, err
		}
		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "pending friend requests rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return requests, nil
}

func (r *Repository) UpdateRequestStatus(ctx context.Context, requestID int64, receiverID int64, status string) (bool, error) {
	result, err := r.DB.Exec(ctx,
		`UPDATE friend_requests SET status = $3, updated_at = NOW()
		WHERE id = $1 AND receiver_id = $2 AND status = $4`,
		requestID, receiverID, status, StatusPending,
	)
	if err != nil {
		slog.ErrorContext(ctx, "update friend request status failed", "request_id", requestID, "error", err)
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *Repository) FindFriends(ctx context.Context, userID int64) ([]Friend, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT u.id, u.username, u.rating, fr.updated_at
		FROM friend_requests fr
		JOIN users u ON u.id = CASE
			WHEN fr.sender_id = $1 THEN fr.receiver_id
			ELSE fr.sender_id
		END
		WHERE (fr.sender_id = $1 OR fr.receiver_id = $1)
			AND fr.status = $2
		ORDER BY u.username`,
		userID, StatusAccepted,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find friends query failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	friends := []Friend{}
	for rows.Next() {
		var friend Friend
		err := rows.Scan(&friend.UserID, &friend.Username, &friend.Rating, &friend.FriendsSince)
		if err != nil {
			slog.ErrorContext(ctx, "scan friend row failed", "user_id", userID, "error", err)
			return nil, err
		}
		friends = append(friends, friend)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "find friends rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return friends, nil
}

func (r *Repository) DeleteSentRequest(ctx context.Context, requestID int64, senderID int64) (bool, error) {
	result, err := r.DB.Exec(ctx,
		`DELETE FROM friend_requests WHERE id = $1 AND sender_id = $2 AND status = $3`,
		requestID, senderID, StatusPending,
	)
	if err != nil {
		slog.ErrorContext(ctx, "delete sent friend request failed", "request_id", requestID, "error", err)
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *Repository) DeleteFriend(ctx context.Context, userID int64, friendID int64) (bool, error) {
	result, err := r.DB.Exec(ctx,
		`DELETE FROM friend_requests
		WHERE status = $3
			AND ((sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1))`,
		userID, friendID, StatusAccepted,
	)
	if err != nil {
		slog.ErrorContext(ctx, "delete friend failed", "user_id", userID, "friend_id", friendID, "error", err)
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *Repository) FindSentRequests(ctx context.Context, userID int64) ([]SentRequest, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT fr.id, u.id, u.username, u.rating, fr.created_at
		FROM friend_requests fr
		JOIN users u ON u.id = fr.receiver_id
		WHERE fr.sender_id = $1 AND fr.status = $2
		ORDER BY fr.created_at DESC`,
		userID, StatusPending,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find sent friend requests failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	requests := []SentRequest{}
	for rows.Next() {
		var request SentRequest
		err := rows.Scan(&request.ID, &request.ReceiverID, &request.ReceiverUsername, &request.ReceiverRating, &request.CreatedAt)
		if err != nil {
			slog.ErrorContext(ctx, "scan sent friend request row failed", "user_id", userID, "error", err)
			return nil, err
		}
		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "sent friend requests rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return requests, nil
}

func (r *Repository) FindUserIDByUsername(ctx context.Context, username string) (int64, error) {
	var userID int64
	err := r.DB.QueryRow(ctx, `SELECT id FROM users WHERE username = $1 AND deleted_at IS NULL`, username).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrUserNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "find user id by username failed", "username", username, "error", err)
		return 0, err
	}
	return userID, nil
}

func (r *Repository) FindPairRequest(ctx context.Context, userID int64, otherID int64) (*PairRequest, error) {
	var request PairRequest
	err := r.DB.QueryRow(ctx,
		`SELECT id, sender_id, status
		FROM friend_requests
		WHERE ((sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1))
			AND status <> $3`,
		userID, otherID, StatusRejected,
	).Scan(&request.ID, &request.SenderID, &request.Status)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "find pair friend request failed", "user_id", userID, "other_id", otherID, "error", err)
		return nil, err
	}
	return &request, nil
}

func (r *Repository) BlockUser(ctx context.Context, blockerID int64, blockedID int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin block user tx failed", "error", err)
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx,
		`INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		blockerID, blockedID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrUserNotFound
		}
		slog.ErrorContext(ctx, "save user block failed", "blocker_id", blockerID, "blocked_id", blockedID, "error", err)
		return err
	}

	_, err = tx.Exec(ctx,
		`DELETE FROM friend_requests
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)`,
		blockerID, blockedID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "delete friend requests on block failed", "blocker_id", blockerID, "blocked_id", blockedID, "error", err)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit block user tx failed", "blocker_id", blockerID, "error", err)
		return err
	}

	return nil
}

func (r *Repository) UnblockUser(ctx context.Context, blockerID int64, blockedID int64) (bool, error) {
	result, err := r.DB.Exec(ctx,
		`DELETE FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2`,
		blockerID, blockedID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "unblock user failed", "blocker_id", blockerID, "blocked_id", blockedID, "error", err)
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (r *Repository) FindBlockedUsers(ctx context.Context, userID int64) ([]BlockedUser, error) {
	rows, err := r.DB.Query(ctx,
		`SELECT u.id, u.username, u.rating, ub.created_at
		FROM user_blocks ub
		JOIN users u ON u.id = ub.blocked_id
		WHERE ub.blocker_id = $1
		ORDER BY ub.created_at DESC`,
		userID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "find blocked users failed", "user_id", userID, "error", err)
		return nil, err
	}
	defer rows.Close()

	blockedUsers := []BlockedUser{}
	for rows.Next() {
		var blockedUser BlockedUser
		err := rows.Scan(&blockedUser.UserID, &blockedUser.Username, &blockedUser.Rating, &blockedUser.BlockedAt)
		if err != nil {
			slog.ErrorContext(ctx, "scan blocked user row failed", "user_id", userID, "error", err)
			return nil, err
		}
		blockedUsers = append(blockedUsers, blockedUser)
	}

	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "blocked users rows iteration error", "user_id", userID, "error", err)
		return nil, err
	}

	return blockedUsers, nil
}

func (r *Repository) IsBlocked(ctx context.Context, blockerID int64, blockedID int64) (bool, error) {
	var blocked bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_blocks WHERE blocker_id = $1 AND blocked_id = $2)`,
		blockerID, blockedID,
	).Scan(&blocked)
	if err != nil {
		slog.ErrorContext(ctx, "check user block failed", "blocker_id", blockerID, "blocked_id", blockedID, "error", err)
		return false, err
	}
	return blocked, nil
}

func (r *Repository) IsBlockedEitherWay(ctx context.Context, userID int64, otherID int64) (bool, error) {
	var blocked bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM user_blocks
			WHERE (blocker_id = $1 AND blocked_id = $2) OR (blocker_id = $2 AND blocked_id = $1)
		)`,
		userID, otherID,
	).Scan(&blocked)
	if err != nil {
		slog.ErrorContext(ctx, "check block between users failed", "user_id", userID, "other_id", otherID, "error", err)
		return false, err
	}
	return blocked, nil
}

func (r *Repository) FindUsernameByID(ctx context.Context, userID int64) (string, error) {
	var username string
	err := r.DB.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, userID).Scan(&username)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "find username by id failed", "user_id", userID, "error", err)
		return "", err
	}
	return username, nil
}

func (r *Repository) FindRequestSenderID(ctx context.Context, requestID int64) (int64, error) {
	var senderID int64
	err := r.DB.QueryRow(ctx, `SELECT sender_id FROM friend_requests WHERE id = $1`, requestID).Scan(&senderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrRequestNotFound
	}
	if err != nil {
		slog.ErrorContext(ctx, "find friend request sender failed", "request_id", requestID, "error", err)
		return 0, err
	}
	return senderID, nil
}

package friends

import (
	"context"
	"errors"
	"log/slog"

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

func (r *Repository) UserExists(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`,
		userID,
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "check user exists failed", "user_id", userID, "error", err)
		return false, err
	}
	return exists, nil
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

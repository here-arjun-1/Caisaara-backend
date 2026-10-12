package invitation

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type InviteRepository interface {
	SaveInvite(ctx context.Context, invite *Invite, ttl time.Duration) error
	GetInvite(ctx context.Context, code string) (*Invite, error)
	GetInviteByGameID(ctx context.Context, gameID string) (*Invite, error)
	DeleteInvite(ctx context.Context, code string) error
	AcquireLock(ctx context.Context, code string, holder int64, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, code string) error
	AddChallenge(ctx context.Context, opponentID int64, code string, ttl time.Duration) error
	GetChallengeCodes(ctx context.Context, userID int64) ([]string, error)
	RemoveChallenge(ctx context.Context, userID int64, code string) error
}

type RedisInviteRepository struct {
	Client *redis.Client
}

func NewRedisInviteRepository(client *redis.Client) *RedisInviteRepository {
	return &RedisInviteRepository{
		Client: client,
	}
}

func inviteKey(code string) string {
	return "invite:" + code
}

func inviteGameKey(gameID string) string {
	return "invite:game:" + gameID
}

func lockKey(code string) string {
	return "invite:lock:" + code
}

func challengesKey(userID int64) string {
	return "invite:challenges:" + strconv.FormatInt(userID, 10)
}

func (r *RedisInviteRepository) SaveInvite(ctx context.Context, invite *Invite, ttl time.Duration) error {
	data, err := json.Marshal(invite)
	if err != nil {
		return err
	}
	pipe := r.Client.Pipeline()
	pipe.Set(ctx, inviteKey(invite.Code), data, ttl)
	pipe.Set(ctx, inviteGameKey(invite.GameID), data, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisInviteRepository) GetInvite(ctx context.Context, code string) (*Invite, error) {
	data, err := r.Client.Get(ctx, inviteKey(code)).Bytes()
	if err != nil {
		return nil, err
	}

	var invite Invite
	if err := json.Unmarshal(data, &invite); err != nil {
		return nil, err
	}

	return &invite, nil
}

func (r *RedisInviteRepository) GetInviteByGameID(ctx context.Context, gameID string) (*Invite, error) {
	data, err := r.Client.Get(ctx, inviteGameKey(gameID)).Bytes()
	if err != nil {
		return nil, err
	}

	var invite Invite
	if err := json.Unmarshal(data, &invite); err != nil {
		return nil, err
	}

	return &invite, nil
}

func (r *RedisInviteRepository) DeleteInvite(ctx context.Context, code string) error {
	invite, _ := r.GetInvite(ctx, code)
	pipe := r.Client.Pipeline()
	pipe.Del(ctx, inviteKey(code))
	if invite != nil && invite.GameID != "" {
		pipe.Del(ctx, inviteGameKey(invite.GameID))
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisInviteRepository) AcquireLock(ctx context.Context, code string, holder int64, ttl time.Duration) (bool, error) {
	return r.Client.SetNX(ctx, lockKey(code), holder, ttl).Result()
}

func (r *RedisInviteRepository) ReleaseLock(ctx context.Context, code string) error {
	return r.Client.Del(ctx, lockKey(code)).Err()
}

func (r *RedisInviteRepository) AddChallenge(ctx context.Context, opponentID int64, code string, ttl time.Duration) error {
	pipe := r.Client.Pipeline()
	pipe.SAdd(ctx, challengesKey(opponentID), code)
	pipe.Expire(ctx, challengesKey(opponentID), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisInviteRepository) GetChallengeCodes(ctx context.Context, userID int64) ([]string, error) {
	return r.Client.SMembers(ctx, challengesKey(userID)).Result()
}

func (r *RedisInviteRepository) RemoveChallenge(ctx context.Context, userID int64, code string) error {
	return r.Client.SRem(ctx, challengesKey(userID), code).Err()
}

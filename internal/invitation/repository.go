package invitation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type InviteRepository interface {
	SaveInvite(ctx context.Context, invite *Invite, ttl time.Duration) error
	GetInvite(ctx context.Context, code string) (*Invite, error)
	DeleteInvite(ctx context.Context, code string) error
	AcquireLock(ctx context.Context, code string, holder int64, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, code string) error
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

func lockKey(code string) string {
	return "invite:lock:" + code
}

func (r *RedisInviteRepository) SaveInvite(ctx context.Context, invite *Invite, ttl time.Duration) error {
	data, err := json.Marshal(invite)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, inviteKey(invite.Code), data, ttl).Err()
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

func (r *RedisInviteRepository) DeleteInvite(ctx context.Context, code string) error {
	return r.Client.Del(ctx, inviteKey(code)).Err()
}

func (r *RedisInviteRepository) AcquireLock(ctx context.Context, code string, holder int64, ttl time.Duration) (bool, error) {
	return r.Client.SetNX(ctx, lockKey(code), holder, ttl).Result()
}

func (r *RedisInviteRepository) ReleaseLock(ctx context.Context, code string) error {
	return r.Client.Del(ctx, lockKey(code)).Err()
}

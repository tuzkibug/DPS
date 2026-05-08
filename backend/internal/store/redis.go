package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"dns-sender/pkg/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisStore{client: client}, nil
}

func (r *RedisStore) key(taskID uuid.UUID, field string) string {
	return fmt.Sprintf("task:%s:%s", taskID.String(), field)
}

func (r *RedisStore) SetTaskStats(ctx context.Context, stats *models.TaskStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, r.key(stats.TaskID, "stats"), data, 24*time.Hour).Err()
}

func (r *RedisStore) GetTaskStats(ctx context.Context, taskID uuid.UUID) (*models.TaskStats, error) {
	data, err := r.client.Get(ctx, r.key(taskID, "stats")).Bytes()
	if err != nil {
		return nil, err
	}

	var stats models.TaskStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *RedisStore) IncrSentCount(ctx context.Context, taskID uuid.UUID, count int64) error {
	key := r.key(taskID, "sent_count")
	return r.client.IncrBy(ctx, key, count).Err()
}

func (r *RedisStore) IncrFailedCount(ctx context.Context, taskID uuid.UUID, count int64) error {
	key := r.key(taskID, "failed_count")
	return r.client.IncrBy(ctx, key, count).Err()
}

func (r *RedisStore) GetSentCount(ctx context.Context, taskID uuid.UUID) (int64, error) {
	key := r.key(taskID, "sent_count")
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (r *RedisStore) GetFailedCount(ctx context.Context, taskID uuid.UUID) (int64, error) {
	key := r.key(taskID, "failed_count")
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (r *RedisStore) SetTaskStatus(ctx context.Context, taskID uuid.UUID, status models.TaskStatus) error {
	return r.client.Set(ctx, r.key(taskID, "status"), string(status), 24*time.Hour).Err()
}

func (r *RedisStore) GetTaskStatus(ctx context.Context, taskID uuid.UUID) (models.TaskStatus, error) {
	val, err := r.client.Get(ctx, r.key(taskID, "status")).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return models.TaskStatus(val), nil
}

func (r *RedisStore) SetStartTime(ctx context.Context, taskID uuid.UUID, t time.Time) error {
	return r.client.Set(ctx, r.key(taskID, "start_time"), t.Format(time.RFC3339), 24*time.Hour).Err()
}

func (r *RedisStore) GetStartTime(ctx context.Context, taskID uuid.UUID) (time.Time, error) {
	val, err := r.client.Get(ctx, r.key(taskID, "start_time")).Result()
	if err == redis.Nil {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, val)
}

func (r *RedisStore) ClearTaskData(ctx context.Context, taskID uuid.UUID) error {
	keys := []string{
		r.key(taskID, "stats"),
		r.key(taskID, "sent_count"),
		r.key(taskID, "failed_count"),
		r.key(taskID, "status"),
		r.key(taskID, "start_time"),
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisStore) Close() error {
	return r.client.Close()
}
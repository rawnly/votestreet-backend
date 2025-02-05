package storage

import (
	"context"
	"sync"

	"github.com/gofiber/storage/redis/v3"
)

var (
	storageMap = make(map[int]redis.Storage)
	mu         sync.Mutex
)

func Redis(db int) *redis.Storage {
	if storage, ok := storageMap[db]; ok {
		return &storage
	}

	storage := redis.New(redis.Config{
		Host:     "localhost",
		Port:     6379,
		Database: db,
	})

	mu.Lock()
	storageMap[db] = *storage
	mu.Unlock()

	return Redis(db)
}

func IsRedisHealthy(ctx context.Context) bool {
	cmd := Redis(0).Conn().Ping(ctx)

	return cmd.Err() == nil
}

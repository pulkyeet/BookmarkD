package cache

import (
	"time"
	"os"
	"fmt"
	"context"
	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	ctx = context.Background()
)

const (
	TTLTrending = 1 * time.Hour
	TTLPopular = 1 * time.Hour
	TTLBookDetails = 24 * time.Hour
	TTLGenres = 24 * time.Hour
	TTLUserFeed = 15 * time.Minute
	TTLUserProfile = 30 * time.Minute
	TTLUserRatings = 30 * time.Minute
	TTLBooksList = 1 * time.Hour
)

func InitRedis() error {
	url := os.Getenv("UPSTASH_REDIS_REST_URL")
	token := os.Getenv("UPSTASH_REDIS_REST_TOKEN")
	if url == "" || token == "" {
		return fmt.Errorf("Upstash redis rest url/token missing")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("Failed to parse redis URL: %w", err)
	}
	opt.Password = token
	Client = redis.NewClient(opt)
	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Failed to connect to redis: %w", err)
	}
	fmt.Println("Redis connected successfully")
	return nil
}

func Get(key string) (string, error) {
	val, err := Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func Set(key, value string, ttl time.Duration) error {
	return Client.Set(ctx, key, value, ttl).Err()
}

func Delete(key string) error {
	return Client.Del(ctx, key).Err()
}

func DeletePattern(pattern string) error {
	iter := Client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := Client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func GenerateKey(path string, userID ...string) string {
	if len(userID) > 0 && userID[0] != "" {
		return fmt.Sprintf("cache:user:%s:%s", userID[0], path)
	}
	return fmt.Sprintf("cache:global:%s", path)
}
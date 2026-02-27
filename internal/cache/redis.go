package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	restURL   string
	restToken string
	ctx       = context.Background()
)

const (
	TTLTrending    = 1 * time.Hour
	TTLPopular     = 1 * time.Hour
	TTLBookDetails = 24 * time.Hour
	TTLGenres      = 24 * time.Hour
	TTLUserFeed    = 15 * time.Minute
	TTLUserProfile = 30 * time.Minute
	TTLUserRatings = 30 * time.Minute
	TTLBooksList   = 1 * time.Hour
)

func InitRedis() error {
	restURL = os.Getenv("UPSTASH_REDIS_REST_URL")
	restToken = os.Getenv("UPSTASH_REDIS_REST_TOKEN")
	if restURL == "" || restToken == "" {
		return fmt.Errorf("Upstash redis rest url/token missing")
	}
	// Ping via REST
	_, err := upstashCmd("PING")
	if err != nil {
		return fmt.Errorf("failed to ping Upstash: %w", err)
	}
	fmt.Println("Redis connected successfully")
	return nil
}

// upstashCmd sends a command to Upstash REST API and returns the result field.
func upstashCmd(args ...interface{}) (interface{}, error) {
	body, _ := json.Marshal(args)
	req, _ := http.NewRequest("POST", restURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+restToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Result interface{} `json:"result"`
		Error  string      `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("upstash error: %s", result.Error)
	}
	return result.Result, nil
}

func Get(key string) (string, error) {
	val, err := upstashCmd("GET", key)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil // cache miss
	}
	return fmt.Sprintf("%v", val), nil
}

func Set(key, value string, ttl time.Duration) error {
	seconds := int(ttl.Seconds())
	_, err := upstashCmd("SET", key, value, "EX", seconds)
	return err
}

func Delete(key string) error {
	_, err := upstashCmd("DEL", key)
	return err
}

func DeletePattern(pattern string) error {
	// SCAN + DEL via REST
	val, err := upstashCmd("SCAN", 0, "MATCH", pattern, "COUNT", 100)
	if err != nil {
		return err
	}
	arr, ok := val.([]interface{})
	if !ok || len(arr) < 2 {
		return nil
	}
	keys, ok := arr[1].([]interface{})
	if !ok {
		return nil
	}
	for _, k := range keys {
		if _, err := upstashCmd("DEL", fmt.Sprintf("%v", k)); err != nil {
			return err
		}
	}
	return nil
}

func GenerateKey(path string, userID ...string) string {
	if len(userID) > 0 && userID[0] != "" {
		return fmt.Sprintf("cache:user:%s:%s", userID[0], path)
	}
	return fmt.Sprintf("cache:global:%s", path)
}
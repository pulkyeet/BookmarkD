package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// ConnectFromEnv tries DATABASE_URL first, falls back to individual vars
func ConnectFromEnv() (*sql.DB, error) {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		log.Println("Connecting to database via DATABASE_URL")
		return connectWithURL(dbURL)
	}

	log.Println("Connecting to database via individual env vars")
	port := 5432
	if p := os.Getenv("DB_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	cfg := Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}
	return Connect(cfg)
}

func connectWithURL(dbURL string) (*sql.DB, error) {
	// Fly.io internal URLs use sslmode=disable
	u, err := url.Parse(dbURL)
	if err == nil {
		q := u.Query()
		if q.Get("sslmode") == "" {
			q.Set("sslmode", "disable")
			u.RawQuery = q.Encode()
			dbURL = u.String()
		}
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	log.Println("Database connected successfully")
	return db, nil
}

func Connect(cfg Config) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	log.Println("Database connected successfully")
	return db, nil
}
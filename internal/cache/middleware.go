package cache

import (
	"bytes"
	"fmt"
	"strconv"
	"net/http"
	"time"

	"github.com/pulkyeet/BookmarkD/internal/middleware"
)

type responseWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
	statusCode int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func CacheMiddleware(ttl time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next(w, r)
				return
			}
			var cacheKey string
			claims, ok := middleware.GetUserFromContext(r)
			if ok {
				cacheKey = GenerateKey(r.URL.Path + r.URL.RawQuery, strconv.Itoa(claims.UserID))
			} else {
				cacheKey = GenerateKey(r.URL.Path + r.URL.RawQuery)
			}
			cached, err := Get(cacheKey)
			if err == nil && cached != "" {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(cached))
				return
			}
			rw := &responseWriter{
				ResponseWriter: w,
				body: &bytes.Buffer{},
				statusCode: http.StatusOK,
			}
			next(rw, r)
			if rw.statusCode == http.StatusOK {
				Set(cacheKey, rw.body.String(), ttl)
			}
			w.Header().Set("X-Cache", "MISS")
		}
	}
}

func InvalidateUserCache(userID string) error {
	pattern := fmt.Sprintf("cache:user:%s:*", userID)
	return DeletePattern(pattern)
}

func InvalidateBookCache(bookID string) error {
	Delete(GenerateKey("/api/books/" + bookID))
	DeletePattern("cache:global:/api/books/trending*")
	DeletePattern("cache:global:/api/books/popular*")
	DeletePattern("cache:global:/api/books?*")
	return nil
}

func InvalidateBookCacheByID(bookID int) error {
	return InvalidateBookCache(strconv.Itoa(bookID))
}
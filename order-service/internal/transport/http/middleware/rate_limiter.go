package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	redisClient *redis.Client
	limit       int
	window      time.Duration
}

func NewRateLimiter(redisClient *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redisClient: redisClient,
		limit:       limit,
		window:      window,
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if rl.redisClient == nil {
			c.Next()
			return
		}

		// Get client IP (handle X-Forwarded-For for proxies)
		clientIP := c.ClientIP()
		if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}

		key := "ratelimit:" + clientIP

		ctx := context.Background()
		now := time.Now().Unix()

		// Use Redis pipeline for atomic operations
		pipe := rl.redisClient.Pipeline()

		// Remove old requests outside the window
		minTime := now - int64(rl.window.Seconds())
		pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(minTime, 10))

		// Add current request
		pipe.ZAdd(ctx, key, &redis.Z{Score: float64(now), Member: strconv.FormatInt(now, 10)})

		// Count requests in window
		pipe.ZCard(ctx, key)

		// Set expiration on the key
		pipe.Expire(ctx, key, rl.window)

		cmds, err := pipe.Exec(ctx)
		if err != nil {
			c.Next() // Allow request if Redis fails
			return
		}

		requestCount := cmds[2].(*redis.IntCmd).Val()

		// Check if limit exceeded
		if requestCount > int64(rl.limit) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": int(rl.window.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

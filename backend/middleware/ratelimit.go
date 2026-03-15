package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	tokens   map[string]*bucket // key = "apiKeyID:endpoint"
	interval time.Duration
}

type bucket struct {
	count     int
	limit     int
	lastReset time.Time
	lastSeen  time.Time
}

var limiter = &rateLimiter{
	tokens:   make(map[string]*bucket),
	interval: time.Minute,
}

func init() {
	// 定期清理过期的限流器条目
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			limiter.mu.Lock()
			for k, b := range limiter.tokens {
				if time.Since(b.lastSeen) > 10*time.Minute {
					delete(limiter.tokens, k)
				}
			}
			limiter.mu.Unlock()
		}
	}()
}

// RateLimit 速率限制中间件（基于 API Key + 端点类型）
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		keyID, exists := c.Get("api_key_id")
		if !exists {
			c.Next()
			return
		}

		rateLimit, _ := c.Get("rate_limit")
		limit, ok := rateLimit.(int)
		if !ok || limit <= 0 {
			limit = 100
		}

		apiKeyID, _ := keyID.(int64)

		// 按端点分类：写入操作限流更严格
		endpoint := "read"
		path := c.Request.URL.Path
		if c.Request.Method == "POST" || c.Request.Method == "DELETE" {
			endpoint = "write"
		}
		_ = path // 未来可按路径进一步细分

		bucketKey := fmt.Sprintf("%d:%s", apiKeyID, endpoint)

		limiter.mu.Lock()
		b, exists := limiter.tokens[bucketKey]
		if !exists || time.Since(b.lastReset) > limiter.interval {
			b = &bucket{count: 0, limit: limit, lastReset: time.Now(), lastSeen: time.Now()}
			limiter.tokens[bucketKey] = b
		}
		b.count++
		b.lastSeen = time.Now()
		exceeded := b.count > b.limit
		limiter.mu.Unlock()

		if exceeded {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"limit": limit,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

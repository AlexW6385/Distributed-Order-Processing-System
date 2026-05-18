package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
	IdleTTL           time.Duration
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimit(config RateLimitConfig) gin.HandlerFunc {
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 2
	}
	if config.Burst <= 0 {
		config.Burst = 10
	}
	if config.IdleTTL <= 0 {
		config.IdleTTL = 10 * time.Minute
	}

	var mu sync.Mutex
	clients := map[string]*clientLimiter{}

	go func() {
		ticker := time.NewTicker(config.IdleTTL)
		defer ticker.Stop()
		for now := range ticker.C {
			mu.Lock()
			for ip, client := range clients {
				if now.Sub(client.lastSeen) > config.IdleTTL {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		client, ok := clients[ip]
		if !ok {
			client = &clientLimiter{
				limiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst),
			}
			clients[ip] = client
		}
		client.lastSeen = now
		allowed := client.limiter.Allow()
		mu.Unlock()

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}

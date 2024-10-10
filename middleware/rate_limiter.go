package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var (
	ratePerMinuteConfig = map[string]int64{
		"1234": 5,
		"2345": 10,
	}
)

type RedisRateLimiter struct {
	client *redis.Client
	mu     *sync.RWMutex
}

func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		mu:     &sync.RWMutex{},
	}
}

func (rr *RedisRateLimiter) Lock(ctx context.Context, key string) (bool, error) {
	return rr.client.SetNX(ctx, fmt.Sprintf("%s-lock", key), 1, 5*time.Minute).Result()
}

func (rr *RedisRateLimiter) AddLimiter(ctx context.Context, key string, expire time.Duration) error {
	val, err := rr.client.Incr(ctx, key).Result()

	if val == 1 {
		err = rr.client.Expire(ctx, key, expire).Err()
		if err != nil {
			defer func() {
				rr.client.Decr(ctx, key).Result()
			}()
		}
	}

	return err
}

func (rr *RedisRateLimiter) IsBellowLimit(ctx context.Context, key string, limit int64) bool {
	valStr, err := rr.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return false
	}

	val, _ := strconv.ParseInt(valStr, 10, 64)

	return val < limit
}

func (rr *RedisRateLimiter) Unlock(ctx context.Context, key string) error {
	return rr.client.Del(ctx, fmt.Sprintf("%s-lock", key)).Err()
}

func (rr *RedisRateLimiter) RateLimitChecker() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		ipAddr := c.ClientIP()
		apiKey := c.GetHeader("x-api-key")
		limiterKey := fmt.Sprintf("%s:%s", apiKey, ipAddr)

		rr.mu.Lock()
		defer rr.mu.Unlock()

		// for !isSuccess {
		isSuccess, err := rr.Lock(ctx, limiterKey)
		if err != nil {
			log.Error().Err(err).Str("ip_address", ipAddr).Str("api_key", apiKey).Msg("can't lock limiter key")
			c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{"message": "cannot process this request"})
			return
		}

		if !isSuccess {
			// time.Sleep(100 * time.Millisecond)
			log.Error().Err(err).Str("ip_address", ipAddr).Str("api_key", apiKey).Msg("failed to lock limiter key")
			c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{"message": "cannot process this request"})
			return
		}
		// }

		defer rr.Unlock(ctx, limiterKey)

		if !rr.IsBellowLimit(ctx, limiterKey, ratePerMinuteConfig[apiKey]) {
			log.Debug().Str("ip_address", ipAddr).Str("api_key", apiKey).Msg("too many request")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, map[string]string{"message": http.StatusText(http.StatusTooManyRequests)})
			return
		}

		err = rr.AddLimiter(ctx, limiterKey, 1*time.Minute)
		if err != nil {
			log.Error().Err(err).Str("ip_address", ipAddr).Str("api_key", apiKey).Msg("can't add limiter key")
			c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{"message": "cannot process this request"})
			return
		}

		c.Next()
	}
}

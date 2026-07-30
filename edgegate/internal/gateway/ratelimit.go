package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var tokenBucketScript = redis.NewScript(`
local capacity = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local refill = tonumber(ARGV[3])
local tokens = tonumber(redis.call('HGET', KEYS[1], 'tokens') or capacity)
local timestamp = tonumber(redis.call('HGET', KEYS[1], 'timestamp') or now)
tokens = math.min(capacity, tokens + math.max(0, now - timestamp) * refill)
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'timestamp', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(1000 / refill) * 2)
return allowed
`)

type limiter struct {
	redis     *redis.Client
	perSecond float64
	burst     int
}

func newLimiter(client *redis.Client, perSecond float64, burst int) *limiter {
	return &limiter{redis: client, perSecond: perSecond, burst: burst}
}

func (l *limiter) allow(ctx context.Context, key string) (bool, error) {
	result, err := tokenBucketScript.Run(ctx, l.redis, []string{"edgegate:ratelimit:" + key}, l.burst, time.Now().UnixMilli(), l.perSecond/1000).Int()
	if err != nil {
		return false, fmt.Errorf("redis token bucket: %w", err)
	}
	return result == 1, nil
}

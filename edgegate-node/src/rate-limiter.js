const TOKEN_BUCKET_SCRIPT = `
local capacity = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local refill = tonumber(ARGV[3])
local tokens = tonumber(redis.call('HGET', KEYS[1], 'tokens') or capacity)
local timestamp = tonumber(redis.call('HGET', KEYS[1], 'timestamp') or now)
tokens = math.min(capacity, tokens + math.max(0, now - timestamp) * refill)
local allowed = 0
if tokens >= 1 then tokens = tokens - 1 allowed = 1 end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'timestamp', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(1000 / refill) * 2)
return allowed
`;

export class RedisTokenBucket {
  constructor(redis, { requestsPerSecond, burst }) {
    this.redis = redis;
    this.requestsPerSecond = requestsPerSecond;
    this.burst = burst;
  }

  async allow(key) {
    const result = await this.redis.eval(TOKEN_BUCKET_SCRIPT, {
      keys: [`edgegate:ratelimit:${key}`],
      arguments: [String(this.burst), String(Date.now()), String(this.requestsPerSecond / 1000)]
    });
    return Number(result) === 1;
  }
}

import { validateConfig } from './config.js';

const UPDATE_SCRIPT = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local expected = tonumber(ARGV[1])
if current ~= expected then return {0, current} end
local next = current + 1
redis.call('SET', KEYS[1], next)
redis.call('SET', KEYS[2], ARGV[2])
redis.call('PUBLISH', KEYS[3], next)
return {1, next}
`;

export class ConfigStore {
  constructor(redis, options = {}) {
    this.redis = redis;
    this.versionKey = options.versionKey || 'edgegate:config:version';
    this.configKey = options.configKey || 'edgegate:config:document';
    this.channel = options.channel || 'edgegate:config:updates';
    this.subscriber = null;
  }

  async current() {
    const [version, document] = await this.redis.mGet([this.versionKey, this.configKey]);
    return {
      version: Number(version || 0),
      config: document ? JSON.parse(document) : null
    };
  }

  async update(config, expectedVersion) {
    validateConfig(config);
    const result = await this.redis.eval(UPDATE_SCRIPT, {
      keys: [this.versionKey, this.configKey, this.channel],
      arguments: [String(expectedVersion), JSON.stringify(config)]
    });
    const [updated, version] = result.map(Number);
    if (updated !== 1) {
      const error = new Error('configuration version conflict');
      error.statusCode = 409;
      error.currentVersion = version;
      throw error;
    }
    return version;
  }

  async subscribe(onUpdate) {
    this.subscriber = this.redis.duplicate();
    await this.subscriber.connect();
    await this.subscriber.subscribe(this.channel, async version => {
      try {
        const current = await this.current();
        if (current.version !== Number(version) || !current.config) return;
        await onUpdate(current.config, current.version);
      } catch (error) {
        console.error(JSON.stringify({ level: 'error', event: 'config_subscription_error', message: error.message }));
      }
    });
  }

  async close() {
    if (this.subscriber?.isOpen) await this.subscriber.quit();
  }
}

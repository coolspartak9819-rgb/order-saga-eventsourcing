import { validateWAFPolicy } from './waf.js';

const UPDATE_SCRIPT = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local expected = tonumber(ARGV[1])
if current ~= expected then return {0, current} end
local next = current + 1
redis.call('SET', KEYS[1], next)
redis.call('SET', KEYS[2], ARGV[2])
redis.call('HSET', KEYS[3], next, ARGV[3])
redis.call('LPUSH', KEYS[4], ARGV[3])
redis.call('LTRIM', KEYS[4], 0, 199)
redis.call('PUBLISH', KEYS[5], next)
return {1, next}
`;

export const DEFAULT_WAF_POLICY = Object.freeze({ mode: 'block', rules: [], disabledDefaultRules: [] });

export class WAFPolicyStore {
  constructor(redis, options = {}) {
    this.redis = redis;
    this.versionKey = options.versionKey || 'edgegate:waf:version';
    this.policyKey = options.policyKey || 'edgegate:waf:policy';
    this.versionsKey = options.versionsKey || 'edgegate:waf:versions';
    this.auditKey = options.auditKey || 'edgegate:waf:audit';
    this.channel = options.channel || 'edgegate:waf:updates';
    this.subscriber = null;
  }

  async current() {
    const [version, document] = await this.redis.mGet([this.versionKey, this.policyKey]);
    return {
      version: Number(version || 0),
      policy: document ? JSON.parse(document) : structuredClone(DEFAULT_WAF_POLICY)
    };
  }

  async update(policy, expectedVersion, metadata = {}) {
    validateWAFPolicy(policy);
    if (!Number.isInteger(expectedVersion) || expectedVersion < 0) throw badRequest('expectedVersion must be a non-negative integer');
    const normalized = normalizePolicy(policy);
    const entry = {
      version: expectedVersion + 1,
      action: metadata.action || 'update',
      actor: normalizeActor(metadata.actor),
      sourceVersion: metadata.sourceVersion ?? null,
      createdAt: new Date().toISOString(),
      policy: normalized
    };
    const result = await this.redis.eval(UPDATE_SCRIPT, {
      keys: [this.versionKey, this.policyKey, this.versionsKey, this.auditKey, this.channel],
      arguments: [String(expectedVersion), JSON.stringify(normalized), JSON.stringify(entry)]
    });
    const [updated, version] = result.map(Number);
    if (updated !== 1) {
      const error = new Error('WAF policy version conflict');
      error.statusCode = 409;
      error.currentVersion = version;
      throw error;
    }
    return version;
  }

  async history(limit = 20) {
    const safeLimit = Math.min(Math.max(Number(limit) || 20, 1), 100);
    return (await this.redis.lRange(this.auditKey, 0, safeLimit - 1)).map(value => JSON.parse(value));
  }

  async rollback(targetVersion, expectedVersion, actor) {
    if (!Number.isInteger(targetVersion) || targetVersion < 1) throw badRequest('targetVersion must be a positive integer');
    const document = await this.redis.hGet(this.versionsKey, String(targetVersion));
    if (!document) {
      const error = new Error('WAF policy version not found');
      error.statusCode = 404;
      throw error;
    }
    const target = JSON.parse(document);
    return this.update(target.policy, expectedVersion, { action: 'rollback', actor, sourceVersion: targetVersion });
  }

  async subscribe(onUpdate) {
    this.subscriber = this.redis.duplicate();
    await this.subscriber.connect();
    await this.subscriber.subscribe(this.channel, async version => {
      try {
        const current = await this.current();
        if (current.version !== Number(version)) return;
        await onUpdate(current.policy, current.version);
      } catch (error) {
        console.error(JSON.stringify({ level: 'error', event: 'waf_policy_subscription_error', message: error.message }));
      }
    });
  }

  async close() {
    if (this.subscriber?.isOpen) await this.subscriber.quit();
  }
}

function normalizePolicy(policy) {
  return {
    mode: policy.mode ?? 'block',
    rules: (policy.rules ?? []).map(rule => ({ id: rule.id, source: rule.source, enabled: rule.enabled !== false })),
    disabledDefaultRules: [...(policy.disabledDefaultRules ?? [])]
  };
}

function normalizeActor(actor) {
  return typeof actor === 'string' && actor.trim() ? actor.trim().slice(0, 128) : 'unknown';
}

function badRequest(message) {
  const error = new Error(message);
  error.statusCode = 400;
  return error;
}

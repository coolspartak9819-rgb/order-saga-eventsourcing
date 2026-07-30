const DEFAULT_RULES = [
  { id: 'sqli', source: String.raw`(union\s+select|select\s+.+\s+from|insert\s+into|drop\s+table|--\s|\/\*.*\*\/)` },
  { id: 'xss', source: String.raw`(<script\b|javascript:|onerror\s*=|onload\s*=)` },
  { id: 'traversal', source: String.raw`(\.\.\/\.\.\/|%2e%2e%2f|\0)` }
];

export class WAF {
  constructor(policy = {}) {
    if (Array.isArray(policy)) policy = { rules: policy };
    validateWAFPolicy(policy);
    this.mode = policy.mode ?? 'block';
    const disabled = new Set(policy.disabledDefaultRules ?? []);
    const rules = [...DEFAULT_RULES.filter(rule => !disabled.has(rule.id)), ...(policy.rules ?? [])];
    this.rules = rules.filter(rule => rule.enabled !== false).map(rule => ({
      id: typeof rule === 'string' ? 'custom' : rule.id,
      regex: new RegExp(typeof rule === 'string' ? rule : rule.source, 'i')
    }));
  }

  inspect({ url, headers, body = '' }) {
    let decodedUrl = url;
    try { decodedUrl = decodeURIComponent(url); } catch {}
    const headerText = Object.entries(headers).map(([key, value]) => `${key}:${value}`).join('\n');
    const input = `${url}\n${decodedUrl}\n${headerText}\n${body}`;
    return this.rules.find(rule => rule.regex.test(input))?.id ?? null;
  }
}

export function validateWAFPolicy(policy) {
  if (!policy || typeof policy !== 'object' || Array.isArray(policy)) throw new Error('WAF policy must be an object');
  if (policy.mode !== undefined && !['block', 'monitor'].includes(policy.mode)) throw new Error('WAF policy mode must be block or monitor');
  if (policy.rules !== undefined && !Array.isArray(policy.rules)) throw new Error('WAF policy rules must be an array');
  if (policy.disabledDefaultRules !== undefined && !Array.isArray(policy.disabledDefaultRules)) {
    throw new Error('disabledDefaultRules must be an array');
  }

  const ids = new Set();
  for (const [index, rule] of (policy.rules ?? []).entries()) {
    if (!rule || typeof rule !== 'object' || typeof rule.id !== 'string' || !rule.id.trim()) {
      throw new Error(`WAF rule ${index} must have an id`);
    }
    if (ids.has(rule.id)) throw new Error(`duplicate WAF rule id: ${rule.id}`);
    ids.add(rule.id);
    if (typeof rule.source !== 'string' || !rule.source) throw new Error(`WAF rule ${rule.id} must have a source`);
    try {
      new RegExp(rule.source, 'i');
    } catch {
      throw new Error(`WAF rule ${rule.id} contains an invalid regular expression`);
    }
  }
}

export async function readBody(request, maxBytes = 1024 * 1024) {
  const chunks = [];
  let size = 0;
  for await (const chunk of request) {
    size += chunk.length;
    if (size > maxBytes) {
      const error = new Error('request body too large');
      error.statusCode = 413;
      throw error;
    }
    chunks.push(chunk);
  }
  return Buffer.concat(chunks);
}

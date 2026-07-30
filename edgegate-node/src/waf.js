const DEFAULT_RULES = [
  { id: 'sqli', source: String.raw`(union\s+select|select\s+.+\s+from|insert\s+into|drop\s+table|--\s|\/\*.*\*\/)` },
  { id: 'xss', source: String.raw`(<script\b|javascript:|onerror\s*=|onload\s*=)` },
  { id: 'traversal', source: String.raw`(\.\.\/\.\.\/|%2e%2e%2f|\0)` }
];

export class WAF {
  constructor(extraRules = []) {
    this.rules = [...DEFAULT_RULES, ...extraRules].map(rule => ({
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

const BUCKETS = [0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10];

export class Metrics {
  constructor() {
    this.requests = new Map();
    this.upstreams = new Map();
    this.wafBlocks = new Map();
    this.rateLimited = new Map();
    this.configReloads = new Map();
    this.durations = new Map();
  }

  recordRequest(route, statusCode, seconds) {
    increment(this.requests, labels({ route, status: statusCode }));
    let histogram = this.durations.get(route);
    if (!histogram) {
      histogram = { count: 0, sum: 0, buckets: BUCKETS.map(() => 0) };
      this.durations.set(route, histogram);
    }
    histogram.count += 1;
    histogram.sum += seconds;
    BUCKETS.forEach((bucket, index) => {
      if (seconds <= bucket) histogram.buckets[index] += 1;
    });
  }

  recordUpstream(backend, statusCode) {
    increment(this.upstreams, labels({ backend, status: statusCode }));
  }

  recordWAFBlock(rule) { increment(this.wafBlocks, labels({ rule })); }
  recordRateLimit(route) { increment(this.rateLimited, labels({ route })); }
  recordConfigReload(status) { increment(this.configReloads, labels({ status })); }

  render(routes) {
    const lines = [
      '# HELP edgegate_requests_total Requests processed by route and status.',
      '# TYPE edgegate_requests_total counter',
      ...renderMap('edgegate_requests_total', this.requests),
      '# HELP edgegate_upstream_requests_total Upstream responses by backend and status.',
      '# TYPE edgegate_upstream_requests_total counter',
      ...renderMap('edgegate_upstream_requests_total', this.upstreams),
      '# HELP edgegate_waf_blocks_total Requests blocked by WAF rule.',
      '# TYPE edgegate_waf_blocks_total counter',
      ...renderMap('edgegate_waf_blocks_total', this.wafBlocks),
      '# HELP edgegate_rate_limited_total Requests rejected by rate limiting.',
      '# TYPE edgegate_rate_limited_total counter',
      ...renderMap('edgegate_rate_limited_total', this.rateLimited),
      '# HELP edgegate_config_reloads_total Distributed configuration reload attempts.',
      '# TYPE edgegate_config_reloads_total counter',
      ...renderMap('edgegate_config_reloads_total', this.configReloads),
      '# HELP edgegate_request_duration_seconds Gateway request latency.',
      '# TYPE edgegate_request_duration_seconds histogram'
    ];

    for (const [route, histogram] of this.durations) {
      BUCKETS.forEach((bucket, index) => lines.push(`edgegate_request_duration_seconds_bucket${labels({ route, le: bucket })} ${histogram.buckets[index]}`));
      lines.push(`edgegate_request_duration_seconds_bucket${labels({ route, le: '+Inf' })} ${histogram.count}`);
      lines.push(`edgegate_request_duration_seconds_sum${labels({ route })} ${histogram.sum}`);
      lines.push(`edgegate_request_duration_seconds_count${labels({ route })} ${histogram.count}`);
    }

    lines.push('# HELP edgegate_backend_healthy Backend health check state.');
    lines.push('# TYPE edgegate_backend_healthy gauge');
    lines.push('# HELP edgegate_backend_active_connections Active proxied requests.');
    lines.push('# TYPE edgegate_backend_active_connections gauge');
    lines.push('# HELP edgegate_circuit_open Circuit breaker open state.');
    lines.push('# TYPE edgegate_circuit_open gauge');
    for (const route of routes) {
      for (const backend of route.balancer.backends) {
        const backendLabels = { route: route.path, backend: backend.url.origin };
        lines.push(`edgegate_backend_healthy${labels(backendLabels)} ${backend.healthy ? 1 : 0}`);
        lines.push(`edgegate_backend_active_connections${labels(backendLabels)} ${backend.active}`);
        lines.push(`edgegate_circuit_open${labels(backendLabels)} ${backend.circuitState === 'open' ? 1 : 0}`);
      }
    }
    return `${lines.join('\n')}\n`;
  }
}

function increment(map, key) { map.set(key, (map.get(key) ?? 0) + 1); }
function renderMap(name, map) { return [...map.entries()].map(([key, value]) => `${name}${key} ${value}`); }
function labels(values) {
  const entries = Object.entries(values).map(([key, value]) => `${key}="${String(value).replaceAll('\\', '\\\\').replaceAll('"', '\\"')}"`);
  return `{${entries.join(',')}}`;
}

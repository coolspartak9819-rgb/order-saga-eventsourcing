import test from 'node:test';
import assert from 'node:assert/strict';
import { Metrics } from '../src/metrics.js';

test('renders Prometheus counters, histogram and backend gauges', () => {
  const metrics = new Metrics();
  metrics.recordRequest('/api', 200, 0.04);
  metrics.recordUpstream('http://backend-a:8080', 200);
  metrics.recordWAFDetection('xss', 'block');
  metrics.recordWAFFalsePositive('xss');
  const output = metrics.render([{
    path: '/api',
    balancer: { backends: [{ url: new URL('http://backend-a:8080'), healthy: true, active: 2, circuitState: 'closed' }] }
  }]);
  assert.match(output, /edgegate_requests_total\{route="\/api",status="200"\} 1/);
  assert.match(output, /edgegate_request_duration_seconds_count\{route="\/api"\} 1/);
  assert.match(output, /edgegate_backend_healthy\{route="\/api",backend="http:\/\/backend-a:8080"\} 1/);
  assert.match(output, /edgegate_waf_detections_total\{rule="xss",mode="block"\} 1/);
  assert.match(output, /edgegate_waf_false_positives_total\{rule="xss"\} 1/);
});

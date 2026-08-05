import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    events: { executor: 'constant-arrival-rate', rate: 100, timeUnit: '1s', duration: '60s', preAllocatedVUs: 20, maxVUs: 100 },
  },
  thresholds: { http_req_failed: ['rate<0.01'], http_req_duration: ['p(95)<250'] },
};

export default function () {
  const id = `load-${__VU}-${__ITER}`;
  const response = http.post(`${__ENV.BASE_URL || 'http://localhost:8080'}/v1/events`, JSON.stringify({
    event_id: id,
    endpoint_id: 'load-test-endpoint',
    endpoint_url: 'https://receiver.example.test/webhook',
    type: 'invoice.created',
    payload: { amount: 1200, source: 'k6' },
  }), { headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'load-test' } });
  check(response, { 'accepted': (r) => r.status === 202 || r.status === 200 });
}

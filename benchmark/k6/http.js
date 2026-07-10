import http from 'k6/http';
import { check } from 'k6';

const HOST = __ENV.HOST || 'localhost';
const PORT = __ENV.HTTP_PORT || '8080';
const API_KEY = __ENV.API_KEY || 'secret';
const EMAIL = __ENV.BENCH_EMAIL || 'bench@example.com';

export const options = {
  vus: parseInt(__ENV.VUS || '50'),
  duration: __ENV.DURATION || '30s',
};

const HEADERS = { headers: { 'X-Api-Key': API_KEY } };

export default function () {
  const res = http.get(
    `http://${HOST}:${PORT}/api/v1/subscriptions?email=${EMAIL}`,
    HEADERS,
  );
  check(res, { 'status 200': (r) => r.status === 200 });
}

import grpc from 'k6/net/grpc';
import { check } from 'k6';

const HOST = __ENV.HOST || 'localhost';
const PORT = __ENV.GRPC_PORT || '9091';
const API_KEY = __ENV.API_KEY || 'secret';
const REPO = __ENV.BENCH_REPO || 'golang/go';
const TAG = __ENV.BENCH_TAG || 'v1.0.0';

const client = new grpc.Client();

export const options = {
  vus: parseInt(__ENV.VUS || '50'),
  duration: __ENV.DURATION || '30s',
};

let connected = false;

export default function () {
  if (!connected) {
    client.connect(`${HOST}:${PORT}`, { plaintext: true, reflect: true });
    connected = true;
  }

  const res = client.invoke(
    'subscription.v1.SubscriptionService/UpdateTag',
    { full_name: REPO, tag: TAG },
    { metadata: { 'x-api-key': API_KEY } },
  );

  check(res, { 'status OK': (r) => r && r.status === grpc.StatusOK });
}
